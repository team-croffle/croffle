# sync-fixtures

Protocol conformance scenarios for Croffle sync, read by **both** sides:

- the server: `apps/server` → `go test ./internal/conformance`
- the desktop: `apps/desktop` → the fixture reader in work item 1.3.0#15

A change to the wire protocol that lands on only one side fails here. The
places that break first are not the obvious ones: second-versus-millisecond
resolution, a column named `recurringRule` whose DTO field is `recurrenceRule`,
and booleans travelling as 0/1.

There is deliberately **no `package.json`**: these are data files, and adding a
workspace package would mean a lockfile entry for something that has no
dependencies and nothing to build.

## Layout

```
scenarios/*.json   one scenario per file
```

## Scenario format

```jsonc
{
  "name": "push-conflict-baseversion",
  "protocol": 1,
  "description": "what this pins down, and why it matters",

  // Accounts and devices the runner creates before the steps.
  "accounts": [{ "id": "acct-1", "username": "alice", "teams": ["team-default"] }],
  "devices": [{ "id": "client-1", "accountId": "acct-1" }],

  "steps": [
    {
      "request": { "endpoint": "/sync/push", "as": "client-1", "body": {} },
      "expect": { "status": 200, "body": {} },
    },
    // Named effects exist for states no request can produce.
    { "effect": "setPurgedThroughSeq", "value": 9 },
  ],

  // Optional, read only by the desktop runner: row encode/decode and the
  // last-writer-wins comparison.
  "rowCodec": {
    "cases": [{ "table": "schedule", "row": {}, "wire": {} }],
    "lww": [{ "a": {}, "b": {}, "winner": "a" }],
  },
}
```

### Matching

`expect.body` is matched as a **subset**, so adding a field to a response does
not break every scenario:

- objects: only the keys named in `expect` are compared, recursively
- arrays: the length must match and elements are compared positionally
- `"$any"` as a value: the key must exist, its value is not compared
- `"$absent"` as a value: the key must **not** exist

### Effects

| effect                | meaning                                           |
| --------------------- | ------------------------------------------------- |
| `setPurgedThroughSeq` | pretend retention has deleted up to that sequence |

### Server-only additions

Some scenarios pin down what a newer server adds without changing protocol 1.
They still say `"protocol": 1` and both runners read them, but only the server
runner checks the steps; the desktop runner reads `rowCodec` only.

| scenario                                 | since server | what it pins down                                                                                                                        |
| ---------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `13-push-conflict-different-fields.json` | 0.1.1        | `differentFields` on a push conflict: sorted, no bookkeeping fields, `[]` for the same data, `deletedAt` for a delete against a live row |

## Adding a scenario

Keep one scenario about one thing, and write the `description` as the sentence
you would want to read in a failure. A scenario that needs a state no endpoint
can produce needs a new effect, and both runners have to implement it.
