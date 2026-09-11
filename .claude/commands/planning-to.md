---
description: Plan every roadmap version up to <version> (inclusive) — plan + work files per version.
argument-hint: <version>
---

Target version: $ARGUMENTS (required; without `v`, e.g. `1.2`).

1. If no argument is given or the version is not a heading in `.ai/ROADMAP.md`, list the available roadmap versions and stop.
2. Walk roadmap versions in semver order from the first unplanned one up to the target. For each: read roadmap + code, write `.ai/plan/<version>_<feature>.md` (`상태: ready`) and `.ai/work/<version>-rc.<N>_<task>.md` (one rc per work item). Skip versions that already have a plan and say so.
3. rc numbering restarts per version; later versions may depend on earlier ones (note it in `depends` as `"<version>-rc.N"`).
4. Report one table per version plus open questions.

Do not implement anything and do not commit.

@.claude/skills/croffle-workflow/SKILL.md
