---
description: Run work items in order up to a target — a work number, a release (<version>-rc.A) or a version — only if it exists in plan + work files.
argument-hint: <N | version_N | version-rc.A | version>
---

Target: $ARGUMENTS (required).

Resolve the target:

- `N` → work items of the current work version up to and including number `N`.
- `<version>_<N>` → the same for that version.
- `<version>-rc.A` → every work item whose `release` is that rc (plus unfinished dependencies).
- `<version>` → every item of that version and of any earlier unfinished version.

If the argument is missing, the version has no `.ai/plan/<version>_*.md`, or the item does not exist under `.ai/work/`, say exactly what is missing (and that `/planning-next` or `/planning-to <version>` creates it) and **stop without doing any work**.

Then run the **Work workflow** for each pending item in dependency order until the target is done. Stop early on a failed gate or a blocked dependency and report where you stopped. Report all commits at the end.

@.claude/skills/croffle-workflow/SKILL.md
