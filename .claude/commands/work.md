---
description: Do the next pending work item (the first todo whose dependencies are done) in the current work version.
---

Optional hint: $ARGUMENTS

1. Find the current work version and its first work file with `status: todo` whose `depends` have all finished (their work files no longer exist), lowest rc.N first. If none: report the version state (all done, or which items are blocked) and stop.
2. Say which item you are taking, then run the **Work workflow** for that single item.
3. Stop after one item. Do not start the next one.

@.claude/skills/croffle-workflow/SKILL.md
