# Repository Guidelines

## Project Structure & Module Organization
- Go CLI core at the root (`main.go`) with packages in:
  - `app/` (TUI), `session/` (`git/`, `tmux/`), `ui/` (views/overlays), `config/`, `daemon/`, `cmd/`, `keys/`, `log/`.
  - Tests co-located as `*_test.go` next to source.
- Web docs site in `web/` (Next.js + TS). Assets in `assets/`.

## Build, Test, and Development Commands
- Go (CLI):
  - `go run .` — run locally.
  - `go build -o cs .` — build binary (use `-ldflags "-X main.version=dev"` when needed).
  - `go test ./...` — run all tests; add `-v` or `-run <Name>` as needed.
  - `golangci-lint run` — run linters (matches CI).
- Web (`web/`):
  - `npm run dev` — local dev server.
  - `npm run build` / `npm start` — production build and serve.
  - `npm run lint` — Next.js ESLint rules.

## Coding Style & Naming Conventions
- Go: format with `gofmt`/`go fmt`; CI enforces formatting and `golangci-lint`.
- Naming: exported identifiers in CamelCase with package-level doc comments; tests as `TestXxx`.
- TypeScript: follow Next.js/ESLint defaults in `web/eslint.config.mjs`; prefer explicit types where helpful.

## Testing Guidelines
- Framework: Go’s `testing` package; keep tests close to code (`*_test.go`).
- Run: `go test -v ./...`; coverage: `go test -cover ./...`.
- Conventions: table-driven tests where appropriate; name files/functions after the unit under test (e.g., `worktree_ops_test.go`, `TestWorktreeOperations`).

## Commit & Pull Request Guidelines
- Commits: concise, imperative subject; prefer Conventional Commits (`feat:`, `fix:`, `refactor:`, `docs:`). Tag releases as `vX.Y.Z` (see `.goreleaser.yaml` and release workflow).
- PRs: include a clear description, testing steps, and screenshots for UI changes; link issues; keep changes focused. Ensure `go test ./...` and `npm run lint` pass and CI is green.

## Security & Configuration Tips
- Never commit secrets. Use env vars (e.g., `OPENAI_API_KEY`) and local config (locate via `cs debug`).
- Prerequisites for running the CLI: `tmux` and `gh` installed (see README).
- When contributing to the web app, avoid adding server secrets; prefer client-safe configuration in `web/`.

