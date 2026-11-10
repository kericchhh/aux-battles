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

## Communication

- When the user says a concept or practice introduced in the conversation is unfamiliar, explain it in plain language, give a task-relevant example, and link to a reliable reference for further learning.

## Validation and handoff

1. Run relevant tests and checks. Use [README.md](README.md) for local and PostgreSQL integration test setup. Report commands, results, and relevant tests that were skipped or could not run.
2. Launch a separate reviewer agent to review the complete task diff against the approved scope. The reviewer must inspect correctness, idiomatic Go, error handling, authentication and authorization, database and media lifecycle, and test coverage where relevant. Report every suspected flaw with its location and likely impact, including concerns that tests did not expose. Wait for a separate reviewer agent if one is unavailable; self-review does not satisfy this step.
3. Show the final change summary, test results, and review findings to the user. Ask the user to validate the changes; treat the work as pending until they explicitly do so.

## Frontend (`web/`)

The web client is a Vite, React, TypeScript, Tailwind CSS v4 and shadcn/ui app in `web/`, managed with pnpm. The working agreement above applies to it unchanged.

- Before proposing a frontend change, consult the official [React](https://react.dev/reference/react), [Tailwind CSS](https://tailwindcss.com/docs), [shadcn/ui](https://ui.shadcn.com/docs), [TanStack Query](https://tanstack.com/query/latest/docs/framework/react/overview) and [React Router](https://reactrouter.com/home) documentation.
- The client calls the API through `/api`, which the Vite dev server proxies to `http://localhost:8080` (override with `API_URL`). Keep requests same-origin; the API uses `SameSite=Lax` session cookies and no CORS.
- Server state lives in TanStack Query; do not add a global store without approval.
- Styling:
  - Design tokens live in `web/src/styles/theme.css`; use them, never raw hex values in components.
  - Keep `className` in JSX to short layout utilities (roughly five or fewer). Move a single element's look into a semantic class in `web/src/styles/components.css`, with variants selected by `data-*` attributes; extract a React component for anything larger.
  - The look is simple app UI in the style of shadcn/ui and Vercel: dark only, black and gray surfaces with one blue accent, Geist at a compact scale, slightly rounded corners and sentence-case text. Cards and the navbar have no borders; separate surfaces with lighter fills and the soft shadows defined in `theme.css`. Inputs keep a visible border. Use monospace only for numbers, codes and measurements. Save expressive styling for the game itself.
  - Motion must communicate a state change and respect `prefers-reduced-motion`.
- Accessibility:
  - Use semantic landmarks and one `h1` per page, and keep the skip link working.
  - Every input needs a visible label, with errors linked through `aria-describedby`.
  - Everything must work from the keyboard with a visible focus ring.
  - Announce asynchronous results with live regions.
  - Text must meet WCAG AA contrast, and touch targets must be at least 44px.
- Validation, from `web/`: `pnpm typecheck`, `pnpm lint`, `pnpm test` (includes axe checks), `pnpm format:check` and `pnpm build`. For UI changes, also run the app against a local API and check phone and desktop widths in a browser. Never use the developer's app database for this; use a disposable PostgreSQL container.
- Reviewers additionally check accessibility, responsive layout, loading, empty and error states, and adherence to the styling rules above.
