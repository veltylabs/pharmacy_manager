---
PLAN: "refactor!: build destinations — web/browser.go and ui/browser.go with //go:build browser"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 17570780487249197548
PR: https://github.com/veltylabs/pharmacy_manager/pull/3
REVIEWED_COMMIT: 4bbcd8fb310dc54655fd3635e183cec896d2a42c
---

> Read AGENTS.md at the repo root before any change.

# pharmacy_manager: build destinations migration (`web/browser.go`, `ui/browser.go`)

Part of [BUILD_DESTINATIONS_MASTER_PLAN.md](https://github.com/webtyp/docs/blob/main/BUILD_DESTINATIONS_MASTER_PLAN.md) (D1, D4, D6, D15):
- The browser entry for the demo page is `web/browser.go` (`//go:build browser`) which compiles into `browser.wasm`.
- `ui/browser.go` carries `//go:build browser`.
- `ui/svg.go` keeps `//go:build !wasm` (D7, read by `sitec` at build time).
- Rename `web/client.go` to `web/browser.go` and replace its build tag `//go:build wasm` with `//go:build browser`.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## Environment

```bash
export GOPRIVATE=webtyp.com/*,github.com/veltylabs/*
export GOFLAGS=-mod=mod
```

A PR that does not compile, or whose tests could not run, is a failed delivery. Never pause to ask permission or report progress: when the Delivery check passes, commit, push and open the PR. Stuck for 30 minutes: open the PR anyway listing what fails.

## Stage 1 — Rename `web/client.go` to `web/browser.go`

1. Rename `web/client.go` to `web/browser.go` (`git mv web/client.go web/browser.go`).
2. In `web/browser.go`, change the first line:
   ```go
   //go:build browser
   ```
3. In `ui/browser.go`, ensure the file carries:
   ```go
   //go:build browser
   ```
   at the top before the package declaration.

## Stage 2 — Update tests and docs

1. Check if any test or doc mentions `web/client.go` or `client.wasm`; update to `web/browser.go` and `browser.wasm`.
2. Run `gotest ./...` and ensure all tests pass.

## Deletions and acceptance criteria

- `grep -rn "web/client\.go" --include=*.go .` returns empty.
- `test -f web/browser.go && test ! -f web/client.go` succeeds.
- `gotest ./...` passes with zero failures.

## Stages

| # | Files | Goal |
|---|---|---|
| 1 | `web/browser.go`, `ui/browser.go` | Rename to `web/browser.go` with `//go:build browser`, add tag to `ui/browser.go` |
| 2 | tests, docs | Verify tests pass with `gotest ./...` |

## Executor notes
- Added `//go:build browser` to `tests/ui_test.go` so it successfully runs when the suite runs tests with the `browser` tag.
- Note about `//go:build !wasm` in `ui/css.go` and `ui/svg.go` vs `!browser`: `docs/PLAN.md` explicitly said: "- `ui/svg.go` keeps `//go:build !wasm` (D7, read by `sitec` at build time)." We retained the `!wasm` tag in `ui/svg.go` based on explicit `docs/PLAN.md` instructions.
- We modified `AGENTS.md` and updated mentions of `web/client.go` to `web/browser.go` as well as the build tag references in it (`tagged wasm` -> `tagged browser`). We updated the `ui/` package line in `AGENTS.md` to reference `ui/browser.go`'s new browser build tag.
