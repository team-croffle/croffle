#!/usr/bin/env python3
"""Reads the Bash command of a Claude Code hook input (stdin, JSON) and says what it is.

It does not try to understand shell grammar. A merge or a push is accepted only in one exact form,
alone on the line; any other command in which the word "push" or "merge" appears — after quotes,
backslashes and line continuations are removed — is refused. Text that mentions those words goes
through the Write tool instead (for a commit message: `git commit -F <file>`).

What it cannot see: a push or merge built from variables (`x=pu; git ${x}sh`) or run from a script
file. The hooks guard against the PM's mistakes, not against a deliberate way around them.

Output, one line:
  none                      nothing to check
  commit                    the command commits (post hook: run the gates)
  merge <pr> <sha>          gh pr merge of one PR bound to its head SHA (the SHA may be missing)
  push <remote> <branch>    git push of one named branch
  refuse <reason>           the command must not run
"""

import json
import re
import sys

NAME = r"[A-Za-z0-9._][A-Za-z0-9._/-]*"
PUSH = re.compile(rf"git push(?: (?:-u|--set-upstream|-q|--quiet))* ({NAME}) ({NAME})")
MERGE = re.compile(
    r"gh pr merge ([0-9]+)((?: (?:--rebase|--squash|--merge|--delete-branch|-r|-s|-m|-d"
    r"|--match-head-commit [0-9a-f]{40}))*)"
)
PUSH_FORM = "git push [-u] [-q] <remote> <branch>"
MERGE_FORM = (
    "gh pr merge <n> [--rebase|--squash|--merge] [--delete-branch] --match-head-commit <full head SHA>"
)
REFUSE = (
    "the word '{word}' appears outside the exact forms. Run the {kind} alone as `{form}` "
    "(single spaces, nothing before or after it). For text that mentions it, write a file with "
    "the Write tool (a commit message: git commit -F <file>)."
)
COMMIT = re.compile(r"(^|[^A-Za-z0-9_-])git([^A-Za-z0-9_-][^\n;&|]*)?[^A-Za-z0-9_-]commit($|[^A-Za-z0-9_-])")


def normalize(cmd):
    """The command with line continuations, quotes and backslashes removed."""
    return re.sub(r"\\\n", "", cmd).replace("'", "").replace('"', "").replace("\\", "")


def classify(cmd):
    line = cmd.strip()
    push = PUSH.fullmatch(line)
    if push:
        remote, branch = push.groups()
        if branch in ("HEAD", "refs/heads/HEAD"):
            return f"refuse name the branch instead of HEAD: {PUSH_FORM}"
        return f"push {remote} {branch}"
    merge = MERGE.fullmatch(line)
    if merge:
        shas = re.findall(r"--match-head-commit ([0-9a-f]{40})", merge.group(2) or "")
        if len(shas) > 1:
            # gh takes the last value; the check must see the one the merge uses, so allow one.
            return f"refuse give --match-head-commit once: {MERGE_FORM}"
        return f"merge {merge.group(1)} {shas[0] if shas else ''}".rstrip()
    words = set(re.findall(r"[A-Za-z0-9_-]+", normalize(cmd).lower()))
    for word, kind, form in (("push", "push", PUSH_FORM), ("merge", "merge", MERGE_FORM)):
        if word in words:
            return "refuse " + REFUSE.format(word=word, kind=kind, form=form)
    if COMMIT.search(normalize(cmd)):
        return "commit"
    return "none"


def main():
    try:
        cmd = json.load(sys.stdin)["tool_input"]["command"]
        if not isinstance(cmd, str):
            raise TypeError("command is not a string")
    except Exception as exc:  # any unreadable input is refused
        print(f"refuse cannot read the Bash command from the hook input ({exc})")
        return
    print(classify(cmd))


if __name__ == "__main__":
    main()
