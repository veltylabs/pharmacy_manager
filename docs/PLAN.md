---
PLAN: "refactor!: pharmacy_manager on the crud-by-declaration framework — Allow and Rules in definition.go, persistence by crud, only domain rules in rule files"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 14112497100722200581
PR: https://github.com/veltylabs/pharmacy_manager/pull/12
REVIEWED_COMMIT: 9d5164b330c1e9a36e8af6f58100cdded655e2bc
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work. Commit after every stage.
> Never run `git reset`, `git restore`, `git checkout -- <file>`, `git checkout HEAD <path>` or `git stash` on your own work: a previous session lost its whole migration with a `git checkout HEAD tests`.
> **Publishing:** you publish only by OPENING THE PULL REQUEST (a `git push` from your sandbox does not reach GitHub, and work not submitted is lost when the session ends). Open the PR after Stage 3 with whatever compiles and keep committing to it.
> **Never ask a yes/no question about progress** («am I on the right track?», «shall I proceed?», «does this approach sound correct?»): the answer is always yes. Ask only when a compile error or a signature in this plan is wrong, and name the symbol.
> Never edit `definition_gen.go`. Never write `go.mod`/`go.sum` lines by hand: only `go get` and `go mod tidy` do. No scratch files and no sed/regex/python scripts over the source: edit by hand, file by file, and run `gotest` after each file.

# Plan: `pharmacy_manager` on the crud-by-declaration framework

The previous session of this plan **deleted the stock operations** (`ReceiveStock`, `RegisterEntry`, `RegisterOutput`, `TransferStock`), `queries.go` and `reports.go`, left both a hand-written and a generated `ModelName`, used `storage.IsNotFound` (it does not exist) and its tree did not compile. This version keeps **every exported method** (other modules and apps call them: `CreateProduct`, `UpdateProduct`, `UpsertProduct`, `PatchProducts`, `DeactivateProduct`, `DeleteProducts`, the same for locations, `GetProduct`, `GetLocation`, `GetBatch`, `List*`, `ReceiveStock`, `RegisterEntry`, `RegisterOutput`, `TransferStock`, `ReviewExpiring`, `ProductExists`, `ProductCard`), moves them onto the store, and makes each rule and its method share one function. **Nothing is deleted except what a stage lists with its grep.**

After: tables `PharmacyProduct`, `PharmacyLocation` (full crud with rules), `PharmacyBatch`, `PharmacyStockLevel`, `PharmacyMovement` (read only), a new table `PharmacyOperation` whose `Create` rule runs a stock operation (`kind` receive | entry | output | transfer) inside the transaction, and two flat read models `expiring_stock` and `low_stock`. The module keeps publishing its events (`pharmacy.movement.registered`), the cost facts and the alerts.

Reference module, already migrated (copy its shape): https://github.com/veltylabs/patient_directory/blob/main — `definition.go`, `rule_patient_*.go`, `store.go`, `seed/demo.go`, `tests/setup_test.go`, `tests/crud_test.go`, `tests/demo_test.go`; and https://github.com/veltylabs/device_manager/blob/main (`view.go`, `seed/`).

## Stages

| # | File | What | Files touched |
|---|---|---|---|
| 1 | [PLAN_STAGE_1_TOOLS_DEFINITION.md](PLAN_STAGE_1_TOOLS_DEFINITION.md) | tools, `definition.go`, `errors.go`, `migrate/` | `definition.go`, `errors.go`, `migrate/migrate.go`, `go.mod`, `go.sum` |
| 2 | [PLAN_STAGE_2_MODULE_STORE.md](PLAN_STAGE_2_MODULE_STORE.md) | `Module.store`, `storeFor`, row helpers, translation table | `module.go` (header), `store.go`, `rows.go` |
| 3 | [PLAN_STAGE_3_STOCK_TX.md](PLAN_STAGE_3_STOCK_TX.md) | the stock logic on `crud.Tx` (`stock_tx.go`) | `stock_tx.go` |
| 4 | [PLAN_STAGE_4_RULES.md](PLAN_STAGE_4_RULES.md) | generate; shared checks; the rule files | `checks.go`, `rule_*.go` |
| 5 | [PLAN_STAGE_5_METHODS.md](PLAN_STAGE_5_METHODS.md) | every kept method on the store; `ops.go`, `crud.go` bodies | `module.go`, `crud.go`, `queries.go`, `reports.go`, `alerts.go`, `ops.go` (deleted) |
| 6 | [PLAN_STAGE_6_BROWSER.md](PLAN_STAGE_6_BROWSER.md) | `view.go`, `ui/`, `seed/` demo | `view.go`, `ui/browser.go`, `seed/seed.go`, `seed/demo.go`, `web/` (deleted) |
| 7 | [PLAN_STAGE_7_TESTS_DOCS.md](PLAN_STAGE_7_TESTS_DOCS.md) | tests, `docs/TESTS.md`, README, architecture | `tests/*.go`, `docs/*`, `README.md` |

## Operation mapping (every operation of today has exactly one row)

`<entity>.<verb>` is `router.Operation{Module: ModelName, Entity: "<entity>", Action: model.<Verb>}`; `ModelName` is the constant `definitionc` writes (`"pharmacy"`, the same wire and RBAC prefix as today): never a literal.

| Today | After |
|---|---|
| `list_products`, `get_product` | `pharmacy_product.read` |
| `create_product`, `upsert_product` | `pharmacy_product.create` / `pharmacy_product.update` |
| `update_product`, `deactivate_product` | `pharmacy_product.update` (`Fields: ["is_active"]` to deactivate) |
| `delete_product` | `pharmacy_product.delete` |
| the same six for locations | `pharmacy_location.*` |
| `list_batches`, `list_stock`, `list_movements` | `pharmacy_batch.read`, `pharmacy_stock_level.read`, `pharmacy_movement.read` |
| `receive_stock` / `register_entry` / `register_output` / `transfer_stock` | `pharmacy_operation.create` with `kind` `receive` / `entry` / `output` / `transfer` |
| `list_expiring` | read model `expiring_stock` with filter `{within_days}` |
| `list_low_stock` | read model `low_stock` |

## Delivery check

1. `gotest` ends green on EVERY destination (`wasm` and `browser` included) and «tests index» ✅.
2. `langc sync .` reports zero untranslated keys for `es` and zero reuse warnings.
3. `grep -rn "reg.Operation\|view.MountSave\|view.MountDelete\|router.Context\|rpc.WriteError\|OpListProducts\|storage.IsNotFound" --include=*.go . | grep -v "_test.go\|^./definition_gen.go"` → empty; `grep -n "m\.db" *.go | grep -v "^definition_gen.go"` → only `New`, `storeFor` and the `Sync` call.
4. `ls *.py *.sh commit_message.txt 2>/dev/null` → empty; `grep -c XXXX go.sum` → 0; `test ! -e web`.
5. `git diff --stat main --diff-filter=D` lists ONLY: `ops.go`, `web/browser.go` and `tests/ops_test.go`. `tests/crud_test.go` and `tests/screen_save_test.go` are KEPT and fixed (they test the kept methods and the screens); `queries.go`, `reports.go`, `crud.go`, `alerts.go`, `module.go` stay with new bodies.
