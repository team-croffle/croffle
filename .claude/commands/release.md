---
description: Release the current desktop version via the Croffle Release GitHub workflow after checking preconditions.
argument-hint: [patch|minor|major|rc] [--dry-run]
---

Arguments: $ARGUMENTS (a release type overrides the plan's 릴리스 메모; `--dry-run` forces `dry_run=true`).

Release the current work version (or the next version if all work is done). Run the **Release workflow**: check every precondition and stop with the list of unmet ones if any fail. `rc` releases run without extra approval once the branch is merged and checks are green; a stable release (`patch|minor|major`) requires that the user explicitly asked for it in this conversation — otherwise stop and say so. Report the run URL and the resulting Release assets.

@.claude/skills/croffle-workflow/SKILL.md
