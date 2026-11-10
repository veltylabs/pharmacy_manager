← [Stage 6](PLAN_STAGE_6_BROWSER.md)

# Stage 7 — tests, translations, documentation

## 7.1 What already exists

| File | What to do |
|---|---|
| every `tests/*_test.go` that builds the module with `pharmacymanager.New(…, Deps{…})` | **Keep.** `Deps.TenantID` is now required: add `TenantID: testTenant` to each such literal (`grep -n "Deps{" tests/*.go conflict_internal_test.go`). |
| `tests/crud_test.go`, `ledger_test.go`, `stock_test.go`, `receive_test.go`, `output_test.go`, `events_test.go`, `alerts_test.go`, `reports_test.go`, `tenant_test.go`, `row_test.go`, `seed_test.go`, `migrate_test.go`, `conformance_test.go` | **Keep.** They test the kept methods. Fix only what Stages 1–6 change: `PatchProducts` with an unknown field now answers a field error naming that field (not `<unknown>`); an empty tenant answers `ErrTenantRequired`; a test of `ListMovements` that depended on the order of two movements written in the same nanosecond compares by `MovementType`/`BatchId` instead of position (the store orders by id). Never delete a test. |
| `conflict_internal_test.go` (root, tests the unexported `applyToLevel`) | **Keep**, with `TenantID: "t1"` added to its `Deps` and the two `m.db.Tx(func(tx *orm.DB) error { return applyToLevel(tx, …) })` calls replaced by `m.store.Atomic(func(tx crud.Tx) error { return applyToLevel(tx, …) })` (import `webtyp.com/crud`). |
| `tests/migrate_test.go` | If the `Exec` double of its connection no longer compiles, make it `Exec(string, ...any) (storage.Result, error)` returning `nil, nil` (import `webtyp.com/storage`); never change what a test asserts. The test of the tables list now expects the sixth table `pharmacy_operation` last. |
| `tests/mount_test.go` | **Rewrite** with `TestMount_MatchesDefinition` below (the old test mounted by hand). |
| `tests/ui_test.go` (`browser` tag) | Keep; the `mockCaller` double takes `router.Operation`: `Call(op router.Operation, args model.Encodable, into model.Decodable, done func(err error))`, `Dispatch(op router.Operation, args model.Encodable)`, and `lastOp` is a `router.Operation` (import `webtyp.com/router`). |
| `tests/screen_save_test.go` | Keep and fix: the caller is `crud.LoopbackActingAs(nil, "u1", m)` (not `loopback.ActingAs`); records carry NO `TenantId` (a server field) and no `Id`. |
| `tests/ops_test.go` | **Delete** (it tests the removed handlers `op*`). `grep -rn "mock.Router\|MountOperations(r" tests` → empty. |

## 7.2 New crud tests, `tests/rules_test.go` (package `tests`), with these helpers

```go
package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/time"
)

// call runs one operation of the module as `user` through the same loopback the demo uses and answers the callback error.
func call(user string, pub events.Publisher, mod *pharmacymanager.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: pharmacymanager.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}

// operate creates one pharmacy_operation as u1 and answers the callback error.
func operate(m *pharmacymanager.Module, op *pharmacymanager.PharmacyOperation) error {
	return call("u1", nil, m, "pharmacy_operation", model.Create, &crud.CreateArgs{Records: []model.Model{op}}, nil)
}

// inDays answers the Unix second `days` days from now.
func inDays(days int) time.Unix { return time.NowUnix() + time.Unix(days)*time.Day }
```
(`time.Day` is used exactly as `tests/receive_test.go` already uses it; copy its arithmetic if the types differ.) The module is `m := newModule(t)` (tenant `testTenant`); `newProduct(t, m, "p1")` and `newLocation(t, m, "Store")` of `setup_test.go` create fixtures through the kept methods; `m.ReceiveStock(receive(p.Id, loc.Id, "L1", inDays(30), 10))` of `receive_test.go` receives stock. A record created through `crud` carries NO `Id` and NO server field (`tenant_id`, `actor_id`, `movement_ids`, `created_at`): sending one answers `ErrServerOnlyField`.

| Test (all in `tests/rules_test.go`) | setup → call → expected |
|---|---|
| `TestProduct_CreateThroughCrud` | `call("u1", nil, m, "pharmacy_product", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyProduct{Sku: "p1", Name: "P", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true}}}, nil)` → no error; `m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: testTenant})` has 1 row with `TenantId == testTenant` |
| `TestProduct_CreateRules` | same create with `ForSale: true` and no price → `model.KindOf(err) == model.ErrorInvalid`; with an existing SKU → `model.ErrorConflict` (`ErrSkuExists`) |
| `TestProduct_UpdateZeroesPriceWhenNotForSale` | `p := newProduct(t, m, "p1")` (for sale, price 1000); `pharmacy_product.update` with `&crud.UpdateArgs{IDs: []string{p.Id}, Fields: []string{"for_sale"}, Record: &pharmacymanager.PharmacyProduct{ForSale: false}}` → no error; `m.GetProduct(testTenant, p.Id)` has `Price == 0` and `ForSale == false` |
| `TestProduct_UpdateSkuTaken` | two products `p1`, `p2`; update `p2` with `Fields: []string{"sku"}`, `Record{Sku: "p1"}` → `model.ErrorConflict` |
| `TestProduct_DeleteWithBatchesRefused` | product + location + `m.ReceiveStock(...)`; `pharmacy_product.delete` (`&crud.DeleteArgs{IDs: []string{p.Id}}`) → `model.ErrorInvalid`; `m.GetProduct` still answers it; a product with no stock deletes without error |
| `TestLocation_DeleteWithStockRefused` | the same for `pharmacy_location.delete` → `model.ErrorInvalid`; an empty location deletes |
| `TestLocation_UpdateNameTaken` | locations `A`, `B`; update `B` with `Fields: []string{"name"}`, `Record{Name: "A"}` → `model.ErrorConflict` |
| `TestOperation_ReceiveCreatesBatchAndLevel` | product + location; `operate(m, &PharmacyOperation{Kind: OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})` → no error; `m.ListBatches` has 1 row, `m.ListStock` has 1 level of quantity 10, `m.ListMovements` has 1 movement of type `string(stock.Purchase)`; reading `pharmacy_operation` (`var list pharmacymanager.PharmacyOperationList`; `call("u1", nil, m, "pharmacy_operation", model.Read, &crud.ReadArgs{}, &list)`) gives 1 row with `ActorId == "u1"` and `MovementIds` equal to that movement's id |
| `TestOperation_OutputFirstExpiryFirst` | two receives of 5 each, batches `L5` expiring in 5 days and `L2` in 2 days; output `{Kind: OperationOutput, ProductId, LocationId, Quantity: 3, MovementType: string(stock.Sale)}` → no error; the only movement of type SALE is on the batch `L2` (`m.ListBatches` finds its id by `BatchNumber`), quantity 3 |
| `TestOperation_OutputNeverNegative` | receive 5; output 6 → `model.ErrorInvalid`; `m.ListStock` still has quantity 5 and `m.ListMovements` has 1 movement (atomic: nothing written) |
| `TestOperation_TransferMovesBetweenLocations` | receive 10 into `A`; transfer `{Kind: OperationTransfer, ProductId, LocationId: A.Id, ToLocationId: B.Id, Quantity: 4}` → level of `A` is 6, level of `B` is 4, two more movements (`TRANSFER_OUT` and `TRANSFER_IN`: use `string(stock.TransferOut)` and `string(stock.TransferIn)`) |
| `TestOperation_UnknownKindRejected` | `Kind: "x"` → `failedField(err) == "kind"` |
| `TestOperation_ServerFieldsRefused` | `Kind: OperationReceive`, `ActorId: "x"` → `model.ErrorInvalid`; no batch exists afterwards |
| `TestMovement_NoWriteOperations` | `call("u1", nil, m, "pharmacy_movement", model.Create, …)` → the error text contains `unknown operation` |
| `TestReadModel_LowStock` | product with `MinQuantity: 10` and 5 received → `var list pharmacymanager.LowStockList`; `call(…, "low_stock", model.Read, &crud.ReadArgs{Filter: &pharmacymanager.LowStock{}}, &list)` → 1 row with `Total == 5`; after receiving 20 more, 0 rows |
| `TestReadModel_ExpiringWithinDays` | a batch expiring in 5 days: filter `&pharmacymanager.ExpiringStock{WithinDays: 30}` → 1 row; `WithinDays: 1` → 0 rows |
| `TestTenant_EmptyTenantRefused` | `m.ListProducts(pharmacymanager.ListProductsArgs{})` → `pharmacymanager.ErrTenantRequired` |
| `TestMount_MatchesDefinition` (rewrite `tests/mount_test.go`) | `crud.Mount(nil, nil, nil, m)` returns nil |
| `TestCreate_PublishesChange` | build the module with a broker as `events_test.go` does and pass it as the FIRST argument of `call`; one `pharmacy_product.create` → one event whose `Payload` is a `*crud.Change` (events reach the publisher given to `LoopbackActingAs`, never `Deps.Publisher`) |
| `TestDemo_Valid` (`tests/demo_test.go`, `//go:build wasm`) | copy https://github.com/veltylabs/device_manager/blob/main/tests/demo_test.go replacing the import path by `github.com/veltylabs/pharmacy_manager/seed` |

A rule that refuses is observed through the callback error; assert the field with `failedField` and the kind with `model.KindOf`. `model.Action` of an operation is `model.Create`, `model.Read`, `model.Update`, `model.Delete`.

## 7.3 `docs/TESTS.md`

One row per `func Test…` of the repo (old and new), under the header `| Test | Rule |` and separator `|---|---|`, each row `| [TestName](../tests/file_test.go) | one line rule |` — the link IS the file; **no File column** (it fails the `gotest` tests-index check). Rules in Spanish. `conflict_internal_test.go` is in the root: its row links `../conflict_internal_test.go`.

## 7.4 Translations and documentation (Spanish, `veltylabs`)

1. `langc sync .`: zero untranslated keys for `es` and zero reuse warnings (other languages empty is a warning of the repo, not of this plan). `Err…` constants are not `lang.Text`: they need no key.
2. `README.md` (≤150 lines) with four sections in this order: `## ¿Por qué existe?` (a pharmacy open to the public: medicines, batches with expiry, where they are kept, every movement of stock; **Sin esto vs. Con esto**; first mentions linked to [`crud`](https://github.com/webtyp/crud), [`stock`](https://github.com/webtyp/stock) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)); `## Antes de comenzar` (Go, `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, a `*orm.DB` with the tables of [migrate/migrate.go](migrate/migrate.go), `Deps.IDs`, `Deps.TenantID`, optional `Deps.Publisher`, `Deps.MoneyScope`, `Deps.AlertArea`); `## Guía paso a paso` (Step 1 a complete compilable `main` that imports `"webtyp.com/fmt"`, never stdlib `fmt`, builds the module over `orm.New(mem.New())`, creates a product and a location with `CreateProduct`/`CreateLocation`, receives 10 units with `ReceiveStock` and prints `ListStock`; Step 2 the same through `pharmacy_operation` (link [tests/rules_test.go](tests/rules_test.go)); Step 3 open the demo with `webtyp` at the repo root, link [seed/demo.go](seed/demo.go)); `## Mapa de documentación` (table `Orden | Documento | Qué pregunta responde y cuándo leerlo` with `docs/ARCHITECTURE.md`, `docs/TESTS.md` and each file of `docs/diagrams/`). Include the operations table of `PLAN.md` («Antes» / «Después») and a permissions table (resource, actions: `pharmacy_product`, `pharmacy_location` all four; `pharmacy_batch`, `pharmacy_stock_level`, `pharmacy_movement` read; `pharmacy_operation` create and read; `expiring_stock`, `low_stock` read).
3. `docs/ARCHITECTURE.md`: keep today's text that is still true and bring it to four sections `## 1. Visión general en un minuto`, `## 2. Piezas y estructura` (a table: `definition.go`, `checks.go`, `stock_tx.go`, `rule_*.go`, `module.go`, `crud.go`, `queries.go`, `reports.go`, `alerts.go`, `store.go`, `rows.go`, `view.go`, `ui/`, `seed/`), `## 3. Flujos paso a paso` (recibir, salir por primero-en-caducar-primero-en-salir, transferir; link the diagram), `## 4. Contratos y límites de seguridad` (stock never negative, an expired batch is never sold, why a rule and its method share `stock_tx.go` and `checks.go`, the store sealed to one tenant).
4. `docs/diagrams/STOCK_OPERATION.md`: context header (what triggers it; `tests/rules_test.go`, `tests/output_test.go`, `tests/stock_test.go` verify it), a `flowchart TD` WITHOUT `subgraph` (kind? → receive / entry / output / transfer → checks → write levels and ledger → events, costs, alert) and a numbered walkthrough of each branch and its error (`ErrInsufficientStock`, `ErrBatchExpired`, `ErrExpiryRequired`, `ErrSameLocation`, `ErrOperationKind`).
5. Translate to Spanish every README, `docs/*`, diagram and code comment still in English; every exported symbol without a usage example gets one.
6. Permanent docs never link `docs/PLAN.md` or its stage files.

Run `gotest` and `langc sync .` one last time and read their output, then check the **Delivery check** of `PLAN.md`, commit, push and make sure the pull request is open.
