# Agent working agreement

## Before changing files

1. Inspect the relevant code, tests, and current worktree. Preserve changes already present in the worktree.
2. Before proposing a Go code change, consult the relevant official Go guidance. Start with [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments); use the [current Go documentation](https://go.dev/doc/) for features and APIs those guides do not cover.
3. Present the intended behavior, the files to be changed, and the validation plan to the user. Obtain explicit approval before creating, editing, deleting, or generating files. Obtain new approval before expanding that boundary.

## Implementing approved changes

- Keep edits within the approved boundary. If review or testing reveals another needed change, propose it and wait for approval before editing.
- For SQL or schema changes, change the source in `db/queries/` or `db/migrations/` and regenerate affected `internal/dbgen/` files with `sqlc`; treat generated Go files as outputs.
- When changing song processing, trace effects across the API, worker, database, and media storage. Update tests that cover changed behavior.
- Format changed Go code with `gofmt`.

## Validation and handoff

1. Run relevant tests and checks. Use [README.md](README.md) for local and PostgreSQL integration test setup. Report commands, results, and relevant tests that were skipped or could not run.
2. Launch a separate reviewer agent to review the complete task diff against the approved scope. The reviewer must inspect correctness, idiomatic Go, error handling, authentication and authorization, database and media lifecycle, and test coverage where relevant. Report every suspected flaw with its location and likely impact, including concerns that tests did not expose. Wait for a separate reviewer agent if one is unavailable; self-review does not satisfy this step.
3. Show the final change summary, test results, and review findings to the user. Ask the user to validate the changes; treat the work as pending until they explicitly do so.
