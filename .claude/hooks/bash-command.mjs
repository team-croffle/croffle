#!/usr/bin/env node
// Reads the Bash command of a Claude Code hook input (stdin, JSON) and says what it is.
//
// It does not try to understand shell grammar. A merge or a push is accepted only in one exact
// form, alone on the line; any other command in which the word "push" or "merge" appears — after
// quotes, backslashes and line continuations are removed — is refused. Text that mentions those
// words goes through the Write tool instead (for a commit message: `git commit -F <file>`).
//
// What it cannot see: a push or merge built from variables (`x=pu; git ${x}sh`) or run from a
// script file. The hooks guard against the PM's mistakes, not against a deliberate way around them.
//
// Output, one line:
//   none                      nothing to check
//   commit                    the command commits (post hook: run the gates)
//   merge <pr> <sha>          gh pr merge of one PR bound to its head SHA (the SHA may be missing)
//   push <remote> <branch>    git push of one named branch
//   refuse <reason>           the command must not run
//
// Tests: scripts/test/bash-command.test.mjs.
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const NAME = '[A-Za-z0-9._][A-Za-z0-9._/-]*';
const PUSH = new RegExp(`^git push(?: (?:-u|--set-upstream|-q|--quiet))* (${NAME}) (${NAME})$`);
const MERGE = new RegExp(
  '^gh pr merge ([0-9]+)((?: (?:--rebase|--squash|--merge|--delete-branch|-r|-s|-m|-d' +
    '|--match-head-commit [0-9a-f]{40}))*)$',
);
const PUSH_FORM = 'git push [-u] [-q] <remote> <branch>';
const MERGE_FORM =
  'gh pr merge <n> [--rebase|--squash|--merge] [--delete-branch] --match-head-commit <full head SHA>';
const refuseWord = (word, form) =>
  `the word '${word}' appears outside the exact forms. Run the ${word} alone as \`${form}\` ` +
  '(single spaces, nothing before or after it). For text that mentions it, write a file with ' +
  'the Write tool (a commit message: git commit -F <file>).';
const COMMIT =
  /(^|[^A-Za-z0-9_-])git([^A-Za-z0-9_-][^\n;&|]*)?[^A-Za-z0-9_-]commit($|[^A-Za-z0-9_-])/;

/** The command with line continuations, quotes and backslashes removed. */
function normalize(cmd) {
  return cmd.replace(/\\\n/g, '').replace(/['"\\]/g, '');
}

// Python's str.isspace() characters, the set str.strip() removes. JavaScript's \s differs: it has
// U+FEFF and lacks U+001C..U+001F and U+0085.
const PY_SPACE =
  '\\t\\n\\v\\f\\r\\x1c-\\x20\\x85\\xa0\\u1680\\u2000-\\u200a\\u2028\\u2029\\u202f\\u205f\\u3000';
const EDGE_SPACE = new RegExp(`^[${PY_SPACE}]+|[${PY_SPACE}]+$`, 'g');

/** Python's str.strip(): leading and trailing whitespace as Python counts it. */
function strip(text) {
  return text.replace(EDGE_SPACE, '');
}

/** One-line verdict for a Bash command (see the header). */
export function classify(cmd) {
  const line = strip(cmd);
  const push = PUSH.exec(line);
  if (push) {
    const [, remote, branch] = push;
    if (branch === 'HEAD' || branch === 'refs/heads/HEAD') {
      return `refuse name the branch instead of HEAD: ${PUSH_FORM}`;
    }
    return `push ${remote} ${branch}`;
  }
  const merge = MERGE.exec(line);
  if (merge) {
    const shas = [...(merge[2] ?? '').matchAll(/--match-head-commit ([0-9a-f]{40})/g)].map(
      (m) => m[1],
    );
    if (shas.length > 1) {
      // gh takes the last value; the check must see the one the merge uses, so allow one.
      return `refuse give --match-head-commit once: ${MERGE_FORM}`;
    }
    return `merge ${merge[1]} ${shas[0] ?? ''}`.trimEnd();
  }
  const words = new Set(
    normalize(cmd)
      .toLowerCase()
      .match(/[A-Za-z0-9_-]+/g) ?? [],
  );
  for (const [word, form] of [
    ['push', PUSH_FORM],
    ['merge', MERGE_FORM],
  ]) {
    if (words.has(word)) {
      return `refuse ${refuseWord(word, form)}`;
    }
  }
  if (COMMIT.test(normalize(cmd))) {
    return 'commit';
  }
  return 'none';
}

/** Reads the hook input and returns the verdict; any unreadable input is refused. */
export function classifyHookInput(text) {
  let cmd;
  try {
    cmd = JSON.parse(text).tool_input.command;
    if (typeof cmd !== 'string') {
      throw new TypeError('command is not a string');
    }
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error);
    return `refuse cannot read the Bash command from the hook input (${detail})`;
  }
  return classify(cmd);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.stdout.write(`${classifyHookInput(readFileSync(0, 'utf8'))}\n`);
}
