---
description: Run work items in order up to a target — a work id (NN or <version>/NN) or a version — only if it exists in plan + work files.
argument-hint: <NN | version/NN | version>
---

Target: $ARGUMENTS (required).

Resolve the target:
- `NN` → item NN of the current work version.
- `<version>/NN` → that item.
- `<version>` → every item of that version and of any earlier unfinished version.

If the argument is missing, the version has no `.ai/plan/<version>_*.md`, or the item does not exist under `.ai/work/`, say exactly what is missing (and that `/planning-next` or `/planning-to <version>` creates it) and **stop without doing any work**.

Then run the **Work workflow** for each pending item in dependency order until the target is done. Stop early on a failed gate or a blocked dependency and report where you stopped. Report all commits at the end.

@.claude/skills/croffle-workflow/SKILL.md
