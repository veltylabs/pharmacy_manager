---
PLAN: "test: pharmacy_manager domain tests on the crud framework (part 2 of 2: tests, translations, docs)"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub). Open it as soon as the first tests pass and keep committing to it.
> **Never ask a yes/no question about progress**; ask only when a signature in this plan is wrong, and name the symbol.
> **Before you submit:** `git status` must be clean and `git diff --stat main` must list the files of the stage. A pull request that carries only go.mod, go.sum or docs/PLAN.md is a failed delivery: `git add -A`, commit and submit again.

# Plan (part 2 of 2): `pharmacy_manager` — the domain tests

Part 1 is merged (v0.1.74): the module is migrated to the crud-by-declaration framework (`definition.go`, `rule_*.go`, `stock_tx.go`, `store.go`, `seed/demo.go`). It has no tests of the new rules yet. This part writes them, runs the translations and brings the documentation to the templates.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```
No `replace` in `go.mod`; no `go get`: the versions are already the right ones. Never edit `definition_gen.go`.

## Rules for the executor

1. **Do not write or summarize a plan:** the stage file is the spec. Start now.
2. **Read files by ranges** (`sed -n 1,80p file`); never one huge `cat`.
3. **A field a test uses is the one in `definition_gen.go`:** `grep -n "type <Entity> struct" -A25 definition_gen.go`.
4. **No python or regex over the source:** edit file by file; `gotest` after each test file.

## Stages

| Stage | File | What |
|---|---|---|
| 7 | [PLAN_STAGE_7_TESTS_DOCS.md](PLAN_STAGE_7_TESTS_DOCS.md) | the 19 domain tests (`tests/rules_test.go`), translations, documentation |

## Delivery check

`gotest` passes on every destination; `langc sync .` reports zero untranslated keys and zero reuse warnings; every test of the stage's tables exists and has its row in `docs/TESTS.md` (`| [TestName](../tests/file_test.go) | rule |`). A tree that does not compile or whose tests did not run is a failed delivery.
