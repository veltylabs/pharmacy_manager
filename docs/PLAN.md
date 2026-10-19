---
PLAN: "feat: stock_manager v1 — products and supplies, locations, batches with expiry, append-only stock movements, FEFO output, expiring-stock report, UI and demo"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> This is a **new module**. `gonew` created only `go.mod`, `LICENSE`, `README.md` and a stub `stock_manager.go`.
> The reference implementation to replicate, file by file, is
> [`github.com/veltylabs/item_catalog`](https://github.com/veltylabs/item_catalog) (read its `model.go`, `mcp.go`,
> `view.go`, `ui/`, `seed/`, `web/client.go`, `migrate/` and `tests/` before writing anything). Where this plan
> and item_catalog differ, this plan wins; where this plan is silent, do what item_catalog does.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

## What this module is (and is not)

The clinic runs a **pharmacy open to the public** and uses clinical **supplies** (gloves, syringes, gauze) in its
consultations. Today the stock is an Excel sheet. This module is the system of record for **what the clinic has**:

- a **product** (one entity for medicines and supplies; `kind` tells them apart),
- the **batches** it comes in (batch number + expiry date),
- the **locations** it is kept in (storeroom, pharmacy, a consulting room),
- every **movement** of stock (append-only).

It is **not** a sales, billing or accounting module, and it knows nothing about services, specialties or patients.
Other modules refer to a product by its string id and keep their own snapshot of its name and price. Selling to the
public (the receipt, the payment) is a later wave: here a sale is just an **output movement** of type `SALE` that
carries the id of the document that caused it (`source_document_type` / `source_document_id`, free text this module
never interprets).

Why a module separate from `item_catalog` (services and specialties): different invariants, different owners and
permissions are granted per module (the RBAC resource prefix is the module's `ModelName`).

## Design gate (public API of a new module)

1. **Prior art:** `item_catalog` (module shape); the clinic's previous system (`inventory_location`,
   `inventory_batch`, `inventory_stock`, `inventory_movement`, with the lessons its own reviews recorded: quantity
   is absolute and the sign comes from the movement type; stock is deducted once, at use); Odoo's and ERPNext's
   "one product master with a track-stock switch" (here the switch is not needed: every product is stocked).
2. **Novice-name test:** `Product`, `Location`, `Batch`, `StockLevel`, `Movement`, `ReceiveStock`, `RegisterOutput`,
   `TransferStock`, `ListExpiring`. A reader guesses each.
3. **Complexity ledger:** + 5 entities, + 18 operations, + 3 readers. Nothing deleted (new module).
4. **Where it belongs:** its own repo; the DAG: it depends only on webtyp libraries (never on another veltylabs module).
5. **What it deletes:** nothing.

## Constants, vocabulary and rules (the contract; implement exactly this)

Package name `stockmanager`. `ModelName = "stock"` (wire and RBAC prefix; `ui.ID = "stock"`).

**Product kinds** (`Kind*`): `KindPharmacy = "PHARMACY"`, `KindSupply = "SUPPLY"`.
**Units** (`Unit*`): `UnitPiece = "PIECE"`, `UnitBox = "BOX"`, `UnitTablet = "TABLET"`, `UnitMl = "ML"`, `UnitGram = "GRAM"`.
**Movement types** (`Move*`), with their direction fixed by the type (the stored `quantity` is always **> 0**):

| Type | Direction | Created by |
|---|---|---|
| `MovePurchase = "PURCHASE"` | in | `ReceiveStock` |
| `MoveAdjustIn = "ADJUST_IN"` | in | `RegisterEntry` (inbound correction of a count) |
| `MoveAdjustOut = "ADJUST_OUT"` | out | `RegisterOutput` (write-offs, counts, expired stock) |
| `MoveSale = "SALE"` | out | `RegisterOutput` |
| `MoveInternalUse = "INTERNAL_USE"` | out | `RegisterOutput` (supplies used in a consultation) |
| `MoveTransferOut = "TRANSFER_OUT"` / `MoveTransferIn = "TRANSFER_IN"` | out / in | `TransferStock` only |

`NoBatchNumber = "NO-BATCH"` — the batch number of a supply that has no lot.

### Entities (`model.Definition` literals in `model.go`; every one has `id` (PK) and `tenant_id` (NotNull))

**`product`** (`ProductModel`)

| Field | Type / widget | Rules |
|---|---|---|
| `sku` | `input.Text()`, Label "SKU" | NotNull; letters, digits and `-`; 1–50; unique per tenant → `ErrSkuExists` |
| `name` | `input.Text()`, Label "Name" | NotNull; 1–255 |
| `kind` | radio of `Kind*`, Label "Kind" | NotNull; closed set |
| `unit` | `input.Select()` of `Unit*`, Label "Unit" | NotNull; closed set |
| `for_sale` | `input.Checkbox()`, Label "For sale" | |
| `price` | `input.Decimal()`, Label "Price" | `for_sale` ⇒ `> 0`, else `ErrSaleNeedsPrice`; otherwise stored as 0 |
| `currency` | `input.Text()`, Label "Currency" | `for_sale` ⇒ exactly 3 letters; otherwise empty |
| `controlled` | `input.Checkbox()`, Label "Controlled" | **flag only: no logic depends on it** |
| `barcode` | `input.Text()`, Label "Barcode", `OmitEmpty` | optional; digits, 8–14 |
| `registry_number` | `input.Text()`, Label "Health registry number", `OmitEmpty` | optional; letters, digits, `-` and `/`; 1–30 |
| `is_active` | `input.Checkbox()`, Label "Active" | NotNull |
| `updated_at` | `model.Int()`, `OmitEmpty` | |

**`stock_location`** (`LocationModel`): `name` (`input.Text()`, Label "Name", 1–100, unique per tenant →
`ErrLocationNameExists`), `is_sales_point` (`input.Checkbox()`, Label "Sales point"), `is_active`
(`input.Checkbox()`, Label "Active"), `updated_at`.

**`stock_batch`** (`BatchModel`): `product_id` (Ref → `ProductModel`, NotNull), `batch_number` (Text, 1–50, unique per
product → `ErrBatchExists`), `expiration_date` (Int, Unix seconds UTC; `0` = no expiry), `is_active`, `updated_at`.
Rules: **a `PHARMACY` product's batch must have `expiration_date > 0`** → `ErrExpiryRequired`; `NoBatchNumber` is
valid only for a `SUPPLY` product → `ErrNoBatchOnPharmacy`.

**`stock_level`** (`StockLevelModel`): `location_id` (Ref, NotNull), `batch_id` (Ref, NotNull), `quantity` (Float),
`revision` (Int), `updated_at`. One row per (location, batch), unique. `quantity` is never negative.

**`stock_movement`** (`MovementModel`): `batch_id`, `location_id` (Refs), `movement_type` (`Move*`), `quantity`
(Float, **> 0**), `source_document_type` (Text, optional), `source_document_id` (Text, optional), `note` (Text,
optional, ≤ 255), `created_at` (Int, Unix seconds UTC), `created_by` (Text, optional: the user id from the caller's
context if the router exposes it; otherwise empty). **Append-only: no update, no delete, no operation for either.**
Correcting a mistake is a new movement.

### Errors (`domainError` constants, like item_catalog; status in `statusOf`)

404: `ErrNotFound` ("stock: not found"), `ErrProductNotFound`, `ErrLocationNotFound`, `ErrBatchNotFound`.
409: `ErrSkuExists`, `ErrLocationNameExists`, `ErrBatchExists`, `ErrConflict` ("stock: concurrent update, retry"),
`ErrBatchExpiryMismatch` (receiving into an existing batch number with a different expiry).
400: `ErrInsufficientStock`, `ErrBatchExpired`, `ErrExpiryRequired`, `ErrNoBatchOnPharmacy`, `ErrSaleNeedsPrice`,
`ErrProductInUse` (delete a product that has batches), `ErrLocationInUse` (delete a location that has stock rows),
`ErrSameLocation`, `ErrProductInactive`, `ErrLocationInactive`, `ErrNoRecords`, `ErrNoIds`, plus `ValidationError`
and `decodeError` as in item_catalog. Every message is a constant string prefixed `stock: `; never `errors` from the
stdlib (`fmt.Err` from `webtyp.com/fmt`).

### Operations (names are constants `Op*`; mount in `MountOperations` with `Requires(resource, action)`)

| Constant = wire name | Resource / action | Does |
|---|---|---|
| `OpListProducts = "list_products"` | `product` / Read | filter by `kind`, `active_only`, `limit`, `offset` |
| `OpGetProduct = "get_product"` | `product` / Read | |
| `OpCreateProduct = "create_product"`, `OpUpdateProduct = "update_product"`, `OpUpsertProduct = "upsert_product"` | `product` / Create, Update, Create\|Update | per item_catalog's `CreateItem`/`UpdateItem`/`UpsertItem` |
| `OpDeactivateProduct = "deactivate_product"`, `OpDeleteProduct = "delete_product"` | `product` / Update, Delete | delete only if the product has no batch |
| `OpListLocations = "list_locations"`, `OpUpsertLocation = "upsert_location"`, `OpDeleteLocation = "delete_location"` | `stock_location` | delete only if no `stock_level` row references it |
| `OpListBatches = "list_batches"` | `stock_batch` / Read | by `product_id` (required) |
| `OpReceiveStock = "receive_stock"` | `stock_movement` / Create | see below |
| `OpRegisterEntry = "register_entry"` | `stock_movement` / Create | see below |
| `OpRegisterOutput = "register_output"` | `stock_movement` / Create | see below |
| `OpTransferStock = "transfer_stock"` | `stock_movement` / Create | see below |
| `OpListStock = "list_stock"` | `stock_level` / Read | by optional `product_id`, `location_id`; only rows with `quantity > 0` |
| `OpListMovements = "list_movements"` | `stock_movement` / Read | by optional `product_id`, `batch_id`, `from`, `to`, `limit` (default 100), newest first |
| `OpListExpiring = "list_expiring"` | `stock_level` / Read | see below |

Each op has its own `…Args` model (first field `tenant_id`), like item_catalog's `ListItemsArgs`.

**`ReceiveStock{ProductId, LocationId, BatchNumber, ExpirationDate, Quantity, SourceDocumentType, SourceDocumentId, Note}`**
Finds the batch of that product with that `BatchNumber` or creates it (applying the batch rules; if it exists and its
expiry differs → `ErrBatchExpiryMismatch`), then adds `Quantity` (> 0) to the (location, batch) `stock_level`
(creating it at 0 if missing) and appends a `PURCHASE` movement. Product and location must exist in the tenant and be
active (`ErrProductInactive`, `ErrLocationInactive`). Returns the movement.

**`RegisterOutput{ProductId, LocationId, Quantity, MovementType, BatchId, SourceDocumentType, SourceDocumentId, Note}`**
`MovementType` ∈ {`SALE`, `INTERNAL_USE`, `ADJUST_OUT`}; any other value → `ValidationError`.
- `BatchId` empty → **FEFO**: take stock from the batches of the product in that location whose `quantity > 0` and
  that are **not expired** (`expiration_date == 0 || expiration_date > now`), ordered by `expiration_date` ascending
  with `0` (no expiry) **last**, ties by batch `id` ascending, until `Quantity` is covered; one movement per batch
  used. If the non-expired stock is smaller than `Quantity` → `ErrInsufficientStock` and **nothing is written**.
- `BatchId` given → only that batch. If it is expired and the type is not `ADJUST_OUT` → `ErrBatchExpired`
  (writing expired stock off is exactly what `ADJUST_OUT` is for). If its quantity is smaller → `ErrInsufficientStock`.
- Returns the list of movements.

**`RegisterEntry`** — an **inbound correction** (stock count found more units than the book): `{ProductId, LocationId,
BatchId, Quantity, Note}` adds to an **existing** batch and appends an `ADJUST_IN` movement.

**`TransferStock{ProductId, FromLocationId, ToLocationId, Quantity, Note}`** — same location → `ErrSameLocation`.
Takes `Quantity` out of the source with the same FEFO rule (`TRANSFER_OUT` movements) and puts the same batches into
the destination (`TRANSFER_IN` movements, same quantities), **atomically**.

**`ListExpiring{WithinDays}`** — `stock_level` rows with `quantity > 0` whose batch expires within `WithinDays` days
from now **or already expired** (`expiration_date > 0 && expiration_date <= now + WithinDays*86400`), as an
`ExpiringStock` list `{product_id, product_name, batch_id, batch_number, location_id, quantity, expiration_date,
expired bool}` ordered by `expiration_date` ascending. `WithinDays < 0` → `ValidationError`.

**Atomicity and concurrency.** Each of `ReceiveStock`, `RegisterOutput`, `RegisterEntry` and `TransferStock` writes its
`stock_level` changes and its movements in **one database transaction**. Every `stock_level` update is conditional on
the `revision` read (`WHERE id = ? AND tenant_id = ? AND revision = ?`, then `revision + 1`); a lost race returns
`ErrConflict`. Copy the transaction/revision pattern of `appointment_booking` (`repository.go`,
`UpdateReservationStatusTx`; `AGENTS.md`, «Concurrencia optimista»). Read it first.

**Multi-tenancy.** Every query and every write carries `tenant_id` in the condition; an id of another tenant is
`ErrNotFound`. No operation takes a tenant from anywhere but its args (as item_catalog does).

**Events** (`Deps.Publisher` optional; nil disables): `TopicMovementRegistered = "stock.movement.registered"`
(payload: the movement), published **after** commit, once per movement. No other topic.

**Cross-module readers** (the interfaces other modules will declare are satisfied structurally; do not import them):
`ProductExists(tenantId, productId string) (bool, error)` — exists and active;
`ProductCard(tenantId, productId string) (name, unit string, price float64, currency string, active bool, err error)` —
absent or foreign tenant ⇒ `ErrNotFound`.

## Stage 1 — scaffold

1. Delete the stub `stock_manager.go`.
2. Create `model.go` (the `Definition` literals, the constants, the errors, the topics, `ModelName`), run `ormc` to
   generate `model_orm.go` (never edit it), `go mod tidy`.
3. Dependencies: the same `webtyp.com/*` set as `item_catalog/go.mod` (`fmt`, `model`, `input`, `orm`, `router`,
   `view`, `events`, `ddl`, `time`, `unixid`, `lang`, and for `ui/` and `web/`: `dom`, `layout`, `svg`, `components`,
   `storage`). Use the versions item_catalog's `go.mod` pins today; **no local `replace`**.
4. `migrate/migrate.go`: `Migrate(conn ddl.Execer, c ddl.Compiler) error` creating the five tables in FK order
   (`Product`, `Location`, `Batch`, `StockLevel`, `Movement`), like item_catalog's.

## Stage 2 — the module

`module.go`: `Deps{IDs model.IDGenerator (required), Publisher events.Publisher (optional)}`, `Module`, `New`
(error text `"stock_manager: Deps.IDs is required"`), the service methods named in the operations table, the two readers
(`ProductExists`, `ProductCard`). `mcp.go`/`ops.go`: `ModelName()`, `MountOperations`, the handlers and
`writeError`/`statusOf`, exactly as item_catalog's. `interfaces.go` is **not** needed (this module defines no port).

Time: use `webtyp.com/time` for "now" (Unix seconds). Do **not** add a clock parameter, option or seam for tests: tests
use expiry dates far in the future and in the past.

## Stage 3 — view

`view.go`: `Item()` for `Product` (`Label` = name, `Description` = sku) and `Location` (`Label` = name);
`NewProductView(caller) view.Presenter`, `NewLocationView(caller) view.Presenter`, built like item_catalog's `NewView`
(`view.NewCallerLister` + `view.New`), with titles "Products" and "Locations" as named constants.

## Stage 4 — `ui/`, `seed/`, `web/`

- `ui/module.go`: `ID = "stock"`, `Label = "Farmacia e insumos"`, icons `IconStock = svg.Icon("stock")`,
  `IconProducts = svg.Icon("stock_products")`, `IconLocations = svg.Icon("stock_locations")`.
- `ui/svg.go` (`//go:build !wasm`): one sprite entry per icon, all three with the same simple box geometry
  (`viewBox "0 0 16 16"`, one path): `M2 4l6-3 6 3v8l-6 3-6-3V4zm6-1.7L4.2 4.2 8 6.1l3.8-1.9L8 2.3zM3 5.4v6l4.5 2.2V7.6L3 5.4zm10 0L8.5 7.6v6L13 11.4v-6z`.
  Do not draw other artwork.
- `ui/browser.go`: `Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)`
  returns `platformd.NewGroupUIModule(ID, Label, IconStock, children)` with two children built as in
  `item_catalog/ui/browser.go`: `ID+".products"` ("Productos", a `crudview` over `NewProductView`) and
  `ID+".locations"` ("Ubicaciones", a `crudview` over `NewLocationView`). `NewRecord` for a product:
  `Kind: KindPharmacy, Unit: UnitPiece, IsActive: true`; for a location: `IsActive: true`. **No screen for stock,
  batches or movements in this version** (a later plan designs the entry/exit forms); say so in `docs/ARCHITECTURE.md`.
- `seed/seed.go`: `Load(m *Module, tenantID string) (Data, error)` through the module's own methods: two locations
  ("Bodega" and "Farmacia", the second `is_sales_point`), three products (one `PHARMACY` for sale with price and
  currency `CLP`, one `PHARMACY` not for sale, one `SUPPLY`), one batch received for each with a future expiry
  (the supply uses `NoBatchNumber`).
- `web/client.go` (`//go:build wasm`): the runnable in-browser demo, copy of item_catalog's with `stockmanager`
  (`storage/mem`, `router/loopback`, the seed, `Browser`).
- All user-visible text in `ui/` and `view.go` is `lang.Text` with English keys; run `langc sync` and add the Spanish
  translations in `lang.json` (Productos, Ubicaciones, Farmacia e insumos, Nombre, Tipo, Unidad, Precio, Moneda,
  Controlado, Código de barras, N.º de registro sanitario, Activo, Punto de venta, Para la venta, and every error
  message that reaches the person).

## Stage 5 — tests (`tests/`, external package `stockmanager_test`, `orm.New(mem.New())`, no mocks of the module)

Behaviour and rules only, never the look. Group them in files by feature; do not duplicate a rule in two files.

1. **Products:** create/get/update/deactivate/delete; `ErrSkuExists` (per tenant, not across tenants);
   `ErrSaleNeedsPrice`; delete with a batch → `ErrProductInUse`; foreign tenant → `ErrNotFound`.
2. **Locations:** unique name, delete in use.
3. **Batches and receiving:** pharmacy batch without expiry → `ErrExpiryRequired`; `NoBatchNumber` on pharmacy →
   `ErrNoBatchOnPharmacy`; receiving twice into the same batch adds up; different expiry → `ErrBatchExpiryMismatch`.
4. **FEFO output:** three batches with different expiries (and one with no expiry) — the output drains the earliest
   expiry first, the no-expiry batch last, splits across batches, writes one movement per batch; an expired batch is
   skipped; not enough non-expired stock → `ErrInsufficientStock` **and the stock and movements are unchanged**;
   forced `BatchId` expired → `ErrBatchExpired` unless `ADJUST_OUT`.
5. **Transfer:** moves the same batches, `ErrSameLocation`, insufficient stock leaves both locations unchanged.
6. **Append-only and invariants:** quantity stored is always > 0; stock never negative after any sequence; the sum of
   `list_stock` equals the sum of in-movements minus out-movements for a product (a property test over a short
   scripted sequence).
7. **Concurrency:** a stale `revision` on a `stock_level` update returns `ErrConflict`. The public API cannot force a
   stale revision, so this one test is a **root-level test** (`conflict_internal_test.go`, package `stockmanager`)
   whose first line is `// Root-level test (justified): exercises the unexported stock_level update — the public API cannot force a stale revision.`
8. **`ListExpiring`:** includes expired and soon-to-expire, excludes far-future and empty rows; negative days → error.
9. **Events:** one `stock.movement.registered` per movement, none when `Publisher` is nil, none on a failed operation.
10. **Conformance:** the module satisfies `router.OperationModule`; every op is mounted with `Requires` and `Accepts`
    (copy item_catalog's `conformance_test.go`).
11. **View test** (`tests/stock_view_test.go` + wasm variant as item_catalog's): the group has the two children; the
    product form renders its fields with their labels.

## Stage 6 — docs

`README.md` (quick start, Ops table, readers, key files), `docs/ARCHITECTURE.md` (scope, entities, rules above, the
"no stock screen yet" note, composition-root example), `docs/diagrams/database.md` (Mermaid ERD of the five tables).
Delete nothing else. `docs/PLAN.md` itself is deleted by the workflow, never by you.

## Delivery check

```
gotest                                                  # vet, race, tests and wasm green
grep -rn "errors\." --include=*.go . | grep -v _test    # → empty (no stdlib errors)
grep -n '^replace' go.mod                               # → empty
git diff --stat main                                    # only the files named in the stages
```

## Stages

| # | Stage | Files |
|---|---|---|
| 1 | scaffold | `model.go`, `model_orm.go` (generated), `go.mod`, `migrate/migrate.go` |
| 2 | the module | `module.go`, `mcp.go`/`ops.go` |
| 3 | view | `view.go` |
| 4 | ui, seed, demo | `ui/*`, `seed/seed.go`, `web/client.go`, `lang.json` |
| 5 | tests | `tests/*` |
| 6 | docs | `README.md`, `docs/ARCHITECTURE.md`, `docs/diagrams/database.md` |
