#!/usr/bin/env python3
"""Reads the Bash command of a Claude Code hook input (stdin, JSON) and says what it is.

It splits the command the way the shell does (shlex): words inside quotes are arguments, not
commands, so a commit message that mentions "git push" is not a push. Only a few exact forms of a
merge and a push are allowed; anything else that is a merge or a push is refused rather than guessed.

Output, one line:
  none                      nothing to check
  commit                    a git commit ran (post hook: run the gates)
  merge <pr> [<sha>]        gh pr merge of one PR, bound to a head SHA when given
  push <remote> <branch>    git push of one named branch
  refuse <reason>           the command must not run
"""

import json
import re
import shlex
import sys

MERGE_FORM = (
    "gh pr merge <n> [--rebase|--squash|--merge] [--delete-branch] --match-head-commit <full head SHA>,"
    " alone"
)
PUSH_FORM = "git push [-u] [-q] <remote> <branch>, alone, from the repository root"
SEPARATORS = {";", "&&", "||", "|", "&", "(", ")", ";;", "|&"}
WRAPPERS = {"command", "builtin", "exec", "nohup", "time", "nice", "env", "sudo", "xargs", "timeout"}
SHELLS = {"sh", "bash", "zsh", "dash"}
NAME = re.compile(r"^[A-Za-z0-9._][A-Za-z0-9._/-]*$")


def split_commands(cmd):
    """Simple commands (lists of words) of a command line; None when it cannot be parsed."""
    lexer = shlex.shlex(cmd.replace("\n", " ; "), posix=True, punctuation_chars=";&|()<>")
    lexer.whitespace_split = True
    lexer.commenters = ""
    try:
        tokens = list(lexer)
    except ValueError:
        return None
    commands, current = [], []
    for tok in tokens:
        if tok in SEPARATORS or set(tok) <= set(";&|()"):
            if current:
                commands.append(current)
            current = []
        else:
            current.append(tok)
    if current:
        commands.append(current)
    return commands


def strip_prefix(words):
    """Drops variable assignments and wrapper commands in front of the real command.

    Returns (words, wrapped). A wrapper's own options and arguments cannot be read reliably
    (sudo -u x, timeout 5), so after a wrapper the command starts at the first git or gh word.
    """
    i, wrapped = 0, False
    while i < len(words) and re.match(r"^[A-Za-z_][A-Za-z0-9_]*=", words[i]):
        i, wrapped = i + 1, True
    if i < len(words) and words[i].rsplit("/", 1)[-1] in WRAPPERS:
        rest = words[i + 1 :]
        for k, w in enumerate(rest):
            if w.rsplit("/", 1)[-1] in ("git", "gh"):
                return rest[k:], True
        return [], True
    return words[i:], wrapped


def classify(words, depth=0):
    """What one simple command is: (kind, details)."""
    words, wrapped = strip_prefix(words)
    if wrapped:
        depth += 1
    if not words:
        return ("none", None)
    prog = words[0].rsplit("/", 1)[-1]
    if prog in SHELLS and "-c" in words[1:]:
        inner = words[words.index("-c") + 1] if words.index("-c") + 1 < len(words) else ""
        return classify_line(inner, depth + 1)
    if prog == "eval":
        return classify_line(" ".join(words[1:]), depth + 1)
    if prog == "git":
        result = classify_git(words)
    elif prog == "gh":
        result = classify_gh(words)
    else:
        return ("none", None)
    if depth > 0 and result[0] in ("merge", "push"):
        form = MERGE_FORM if result[0] == "merge" else PUSH_FORM
        return ("refuse", f"a {result[0]} must be the only command, not chained or wrapped: {form}")
    return result


def git_subcommand(words):
    """Index of the git subcommand, skipping global options and their arguments."""
    i = 1
    with_arg = {"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--exec-path", "--config-env"}
    while i < len(words):
        w = words[i]
        if w in with_arg:
            i += 2
        elif w.startswith("-"):
            i += 1
        else:
            return i
    return None


def classify_git(words):
    i = git_subcommand(words)
    if i is None:
        return ("none", None)
    sub = words[i]
    if sub == "commit":
        return ("commit", None)
    if sub != "push":
        return ("none", None)
    if i != 1:
        return ("refuse", f"git options before push are not allowed (no -C, -c, --git-dir): {PUSH_FORM}")
    args = [a for a in words[2:] if a not in ("-u", "--set-upstream", "-q", "--quiet")]
    if len(args) != 2 or not all(NAME.match(a) for a in args):
        return ("refuse", f"a push must name one remote and one branch, with no other option or refspec: {PUSH_FORM}")
    remote, branch = args
    if branch in ("HEAD", "refs/heads/HEAD"):
        return ("refuse", f"name the branch instead of HEAD: {PUSH_FORM}")
    return ("push", (remote, branch))


def classify_gh(words):
    rest = words[1:]
    if len(rest) < 2 or rest[0] != "pr" or rest[1] != "merge":
        # gh with options before "pr" (for example -R) that merges is refused, not guessed.
        if "pr" in rest and "merge" in rest and rest.index("merge") == rest.index("pr") + 1:
            return ("refuse", f"a merge must be exactly: {MERGE_FORM}")
        return ("none", None)
    args = rest[2:]
    if not args or not args[0].isdigit():
        return ("refuse", f"a merge must name the PR number first: {MERGE_FORM}")
    pr, sha = args[0], ""
    allowed = {"--rebase", "--squash", "--merge", "--delete-branch", "-r", "-s", "-m", "-d"}
    j = 1
    while j < len(args):
        a = args[j]
        if a == "--match-head-commit" and j + 1 < len(args) and re.fullmatch(r"[0-9a-f]{40}", args[j + 1]):
            sha = args[j + 1]
            j += 2
        elif a in allowed:
            j += 1
        else:
            return ("refuse", f"a merge must be exactly: {MERGE_FORM}")
    return ("merge", (pr, sha))


def classify_line(line, depth=0):
    if depth > 3:
        return ("refuse", "nested shells are too deep to read; run the command directly")
    commands = split_commands(line)
    if commands is None:
        if re.search(r"\b(push|merge|commit)\b", line):
            return ("refuse", "the command cannot be split into words (unbalanced quotes?)")
        return ("none", None)
    results = [classify(c, depth) for c in commands]
    acting = [r for r in results if r[0] != "none"]
    for r in acting:
        if r[0] == "refuse":
            return r
    guarded = [r for r in acting if r[0] in ("merge", "push")]
    if guarded and (len(commands) > 1 or depth > 0):
        form = MERGE_FORM if guarded[0][0] == "merge" else PUSH_FORM
        return ("refuse", f"a {guarded[0][0]} must be the only command, not chained or wrapped: {form}")
    if guarded:
        return guarded[0]
    if any(r[0] == "commit" for r in acting):
        return ("commit", None)
    return ("none", None)


def main():
    try:
        cmd = json.load(sys.stdin)["tool_input"]["command"]
        if not isinstance(cmd, str):
            raise TypeError
    except Exception:
        print("refuse cannot read the Bash command from the hook input")
        return
    kind, detail = classify_line(cmd)
    if kind == "refuse":
        print(f"refuse {detail}")
    elif kind == "merge":
        print(f"merge {detail[0]} {detail[1]}".rstrip())
    elif kind == "push":
        print(f"push {detail[0]} {detail[1]}")
    else:
        print(kind)


if __name__ == "__main__":
    main()
