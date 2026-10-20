---
PLAN: "feat: pharmacy_manager v0.1.0 — the clinic pharmacy: medicines, batches with mandatory expiry, locations and append-only stock movements, FEFO output over webtyp/stock, cost facts over webtyp/money, UI and demo"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 6796631504616330770
---

> Read AGENTS.md at the repo root before any change.
> This is a **new module**. `gonew` created only `go.mod`, `LICENSE`, `README.md` and a stub `pharmacy_manager.go`.
> The reference implementation to replicate, file by file, is
> [`github.com/veltylabs/item_catalog`](https://github.com/veltylabs/item_catalog) (read its `model.go`, `mcp.go`,
> `view.go`, `ui/`, `seed/`, `web/client.go`, `migrate/` and `tests/` before writing anything). Where this plan
> and item_catalog differ, this plan wins; where this plan is silent, do what item_catalog does.
> Dependencies that are **libraries** (`webtyp.com/stock`, `webtyp.com/money`, `webtyp.com/events`) are published; use their
> exported API exactly as documented in their README and doc comments. If any symbol this plan names is missing from the
> published version, **stop** and say which in the PR.

## Tools

Set these in every shell that runs `go` (the `webtyp.com/*` modules are vanity paths that resolve to GitHub; without them
the public proxy cannot fetch them): `export GOPRIVATE=webtyp.com/*,github.com/veltylabs/*` and `export GOFLAGS=-mod=mod`.
Write `go.mod` with the requires of Stage 1 and run `go mod tidy` **before** running `ormc`. If a module still cannot be
fetched, stop and name the module and the error in the PR; never vendor, copy or hand-write generated code.

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

## What this module is (and is not)

The clinic runs a **pharmacy open to the public**. Today its stock is an Excel sheet. This module is the system of record
for **what the pharmacy has**:

- a **product** (a medicine or any article the pharmacy stocks),
- the **batches** it comes in (batch number, **mandatory expiry date**, optional unit cost),
- the **locations** it is kept in (storeroom, counter),
- every **movement** of stock (append-only).

Clinical **supplies** (gloves, syringes, gauze) are a **different module**, `supply_manager`, with different managers,
permissions and rules; nothing here knows it. This module is **not** a sales, billing or accounting module and knows
nothing about services, specialties or patients: other modules refer to a product by its string id and keep their own
snapshot of its name and price. Selling to the public (the receipt, the payment) is a later wave: here a sale is just an
**output movement** of type `stock.Sale` that carries the id of the document that caused it (`source_document_type` /
`source_document_id`, free text this module never interprets).

Do not invent legal rules this plan does not state: they are validated with the clinic's responsible pharmacist
(`controlled` is only a flag).

## Ecosystem search (done before writing this plan)

| Found | What it offers | Decision |
|---|---|---|
| `webtyp/stock` v0.1.0 | `Movement` (+ `Direction`), `Lot`, `Allocate` (first expiry first out, all-or-nothing), `Apply` (never-negative arithmetic), `Lot.Expired`, `Lot.ExpiresWithin`, and `stocktest.Run` (conformance suite) | **use for every stock rule**. This module writes no FEFO, no movement-type table and no expiry predicate of its own |
| `webtyp/money` v0.1.0 | `Money`, `Currency` (`CLP`), `Fact` (`Consumed`), `Publish(events.Publisher, Fact)` | **use** for the cost fact of an output (see «Cost»). No local money type |
| `webtyp/events` v0.1.0 | `Publisher`, `Event`, `model.Encodable` payloads | **use** (`Deps.Publisher`, optional) |
| `veltylabs/supply_manager` (sibling module) | the same stock concerns for a different area | **never imported** (modules do not import each other). The rules they share are in `webtyp/stock`; the tables are separate on purpose (`ormc` generates each table from a static literal inside its own module) |
| `veltylabs/item_catalog` | module shape: model, ops, view, ui, seed, web, migrate, tests | **replicate** |
| `veltylabs/appointment_booking` | optimistic concurrency with `revision` inside a transaction (`repository.go`, `UpdateReservationStatusTx`) | **replicate** the pattern; read it first |
| `webtyp/notify`, `webtyp/bus` | alerts / a retired pub/sub | not used here: a later module (`alert_manager`) reads `list_low_stock` and `list_expiring` and raises the alerts |

## Design gate (public API of a new module)

1. **Prior art:** `item_catalog` (module shape); the clinic's previous system (`inventory_location`, `inventory_batch`,
   `inventory_stock`, `inventory_movement`; lessons its own reviews recorded: the quantity is absolute and the sign comes
   from the movement type; stock is deducted once, at use); Odoo's and ERPNext's stock ledger (append-only moves, on-hand
   as a sum, cost per lot).
2. **Novice-name test:** `Product`, `Location`, `Batch`, `StockLevel`, `Movement`, `ReceiveStock`, `RegisterOutput`,
   `TransferStock`, `ListExpiring`, `ListLowStock`. A reader guesses each.
3. **Complexity ledger:** + 5 entities, + 19 operations, + 2 readers. Nothing deleted (new module).
4. **Where it belongs:** its own repo; it depends only on webtyp libraries, never on another veltylabs module.
5. **What it deletes:** nothing.

## Constants, vocabulary and rules (the contract; implement exactly this)

Package name `pharmacymanager`. `ModelName = "pharmacy"` (wire and RBAC prefix; `ui.ID = "pharmacy"`).

**Units** (`Unit*`, closed set): `UnitPiece = "PIECE"`, `UnitBox = "BOX"`, `UnitTablet = "TABLET"`, `UnitMl = "ML"`, `UnitGram = "GRAM"`.

**Movement types** are `webtyp.com/stock`'s `stock.Movement` (`stock.Purchase`, `stock.AdjustIn`, `stock.TransferIn`,
`stock.AdjustOut`, `stock.Sale`, `stock.InternalUse`, `stock.TransferOut`) and their direction is `Movement.Direction()`.
The stored `quantity` is always **> 0**; **no local constant duplicates them**. Outputs this module accepts through
`RegisterOutput`: `stock.Sale`, `stock.InternalUse`, `stock.AdjustOut`.

### Entities (`model.Definition` literals in `model.go`; every one has `id` (PK) and `tenant_id` (NotNull))

Table names are prefixed `pharmacy_` so that this module and its sibling never collide in one database.

**`pharmacy_product`** (`ProductModel`)

| Field | Type / widget | Rules |
|---|---|---|
| `sku` | `input.Text()`, Label "SKU" | NotNull; letters, digits and `-`; 1–50; unique per tenant → `ErrSkuExists` |
| `name` | `input.Text()`, Label "Name" | NotNull; 1–255 |
| `unit` | `input.Select()` of `Unit*`, Label "Unit" | NotNull; closed set |
| `tax_treatment` | `input.Select()` of `TaxTaxed = "TAXED"`, `TaxExempt = "EXEMPT"`, Label "Tax treatment" | NotNull; closed set → `ErrTaxTreatmentRequired`. The rate (19 %) is **not** stored here: it is the law's and belongs to the accounting configuration |
| `for_sale` | `input.Checkbox()`, Label "For sale" | |
| `price` | `input.Number()`, `model.Int()`, Label "Price" | **whole Chilean pesos, tax included**; `for_sale` ⇒ `> 0`, else `ErrSaleNeedsPrice`; otherwise stored as 0 |
| `controlled` | `input.Checkbox()`, Label "Controlled" | **flag only: no logic depends on it** |
| `barcode` | `input.Text()`, Label "Barcode", `OmitEmpty` | optional; digits, 8–14 |
| `registry_number` | `input.Text()`, Label "Health registry number", `OmitEmpty` | optional; letters, digits, `-` and `/`; 1–30 |
| `min_quantity` | `input.Number()`, `model.Float()`, Label "Minimum quantity" | `>= 0`; `0` = no low-stock alert |
| `is_active` | `input.Checkbox()`, Label "Active" | NotNull |
| `updated_at` | `model.Int()`, `OmitEmpty` | |

**`pharmacy_location`** (`LocationModel`): `name` (`input.Text()`, Label "Name", 1–100, unique per tenant →
`ErrLocationNameExists`), `is_sales_point` (`input.Checkbox()`, Label "Sales point"), `is_active` (`input.Checkbox()`, Label "Active"), `updated_at`.

**`pharmacy_batch`** (`BatchModel`): `product_id` (Ref → `ProductModel`, NotNull), `batch_number` (Text, 1–50, unique per
product → `ErrBatchExists`), `expiration_date` (Int, Unix seconds UTC; `0` = no expiry), `unit_cost` (Int, **whole Chilean
pesos per unit**, `0` = unknown, never negative), `is_active`, `updated_at`.
Rules: **a batch must have `expiration_date > 0`** → `ErrExpiryRequired`; `NoBatchNumber` does **not** exist in this module
(a medicine always has a lot): a blank batch number is `ValidationError`.

**`pharmacy_stock_level`** (`StockLevelModel`): `location_id` (Ref, NotNull), `batch_id` (Ref, NotNull), `quantity` (Float),
`revision` (Int), `updated_at`. One row per (location, batch), unique. `quantity` is never negative (`stock.Apply`).

**`pharmacy_movement`** (`MovementModel`): `batch_id`, `location_id` (Refs), `movement_type` (`stock.Movement`), `quantity`
(Float, **> 0**), `source_document_type` (Text, optional), `source_document_id` (Text, optional), `note` (Text,
optional, ≤ 255), `created_at` (Int, Unix seconds UTC), `created_by` (Text, optional: the user id from the caller's
context if the router exposes it; otherwise empty). **Append-only: no update, no delete, no operation for either.**
Correcting a mistake is a new movement.

### Errors (`domainError` constants, like item_catalog; status in `statusOf`)

404: `ErrNotFound` ("pharmacy: not found"), `ErrProductNotFound`, `ErrLocationNotFound`, `ErrBatchNotFound`.
409: `ErrSkuExists`, `ErrLocationNameExists`, `ErrBatchExists`, `ErrConflict` ("pharmacy: concurrent update, retry"),
`ErrBatchExpiryMismatch` (receiving into an existing batch number with a different expiry), `ErrBatchCostMismatch`
(receiving into an existing batch with a different non-zero unit cost).
400: `ErrInsufficientStock`, `ErrBatchExpired`, `ErrExpiryRequired`, `ErrSaleNeedsPrice`, `ErrNotForSale` (a `stock.Sale` output of a product that is not for sale), `ErrTaxTreatmentRequired`, `ErrProductInUse` (delete a product that has batches),
`ErrLocationInUse` (delete a location that has stock rows), `ErrSameLocation`, `ErrProductInactive`, `ErrLocationInactive`,
`ErrNoRecords`, `ErrNoIds`, plus `ValidationError` and `decodeError` as in item_catalog. Map `stock.ErrInsufficient` to
`ErrInsufficientStock` and `stock.ErrQuantity` to `ValidationError`. Every message is a constant string prefixed
`pharmacy: `; never `errors` from the stdlib (`fmt.Err` from `webtyp.com/fmt`).

### Operations (names are constants `Op*`; mount in `MountOperations` with `Requires(resource, action)`)

The resource of an operation is the table it works on (`pharmacy_product`, `pharmacy_location`, `pharmacy_batch`, `pharmacy_stock_level`,
`pharmacy_movement`) so that a role can be granted exactly one of them.

| Constant = wire name | Resource / action | Does |
|---|---|---|
| `OpListProducts = "list_products"` | `pharmacy_product` / Read | filter by `active_only`, `limit`, `offset` |
| `OpGetProduct = "get_product"` | `pharmacy_product` / Read | |
| `OpCreateProduct = "create_product"`, `OpUpdateProduct = "update_product"`, `OpUpsertProduct = "upsert_product"` | `pharmacy_product` / Create, Update, Create\|Update | per item_catalog's `CreateItem`/`UpdateItem`/`UpsertItem` |
| `OpDeactivateProduct = "deactivate_product"`, `OpDeleteProduct = "delete_product"` | `pharmacy_product` / Update, Delete | delete only if the product has no batch |
| `OpListLocations = "list_locations"`, `OpUpsertLocation = "upsert_location"`, `OpDeleteLocation = "delete_location"` | `pharmacy_location` | delete only if no stock-level row references it |
| `OpListBatches = "list_batches"` | `pharmacy_batch` / Read | by `product_id` (required) |
| `OpReceiveStock = "receive_stock"` | `pharmacy_movement` / Create | see below |
| `OpRegisterEntry = "register_entry"` | `pharmacy_movement` / Create | see below |
| `OpRegisterOutput = "register_output"` | `pharmacy_movement` / Create | see below |
| `OpTransferStock = "transfer_stock"` | `pharmacy_movement` / Create | see below |
| `OpListStock = "list_stock"` | `pharmacy_stock_level` / Read | by optional `product_id`, `location_id`; only rows with `quantity > 0` |
| `OpListMovements = "list_movements"` | `pharmacy_movement` / Read | by optional `product_id`, `batch_id`, `from`, `to`, `limit` (default 100), newest first |
| `OpListExpiring = "list_expiring"` | `pharmacy_stock_level` / Read | see below |
| `OpListLowStock = "list_low_stock"` | `pharmacy_stock_level` / Read | see below |

Each op has its own `…Args` model (first field `tenant_id`), like item_catalog's `ListItemsArgs`.

**`ReceiveStock{ProductId, LocationId, BatchNumber, ExpirationDate, UnitCost, Quantity, SourceDocumentType, SourceDocumentId, Note}`**
Finds the batch of that product with that `BatchNumber` or creates it (applying the batch rules; if it exists and its
expiry differs → `ErrBatchExpiryMismatch`; if it exists and both costs are non-zero and differ → `ErrBatchCostMismatch`;
a `UnitCost` of 0 into an existing batch keeps the stored cost), then adds `Quantity` (> 0) to the (location, batch)
stock level (creating it at 0 if missing, using `stock.Apply`) and appends a `stock.Purchase` movement. The product and the
location must exist in the tenant and be active (`ErrProductInactive`, `ErrLocationInactive`). Returns the movement.

**`RegisterOutput{ProductId, LocationId, Quantity, MovementType, BatchId, SourceDocumentType, SourceDocumentId, Note}`**
`MovementType` ∈ {`stock.Sale`, `stock.InternalUse`, `stock.AdjustOut`}; any other value → `ValidationError`. A `stock.Sale`
of a product with `for_sale = false` → `ErrNotForSale`. An expired batch is never sold or used (`ErrBatchExpired`); only
`AdjustOut` may take expired stock.
- `BatchId` empty → **first expiry first out** through `stock.Allocate(lots, quantity, now)`: build the `[]stock.Lot` from
  the stock levels of that product in that location (`Lot{ID: batch id, Expiry: expiration_date, Quantity: level quantity}`),
  take what `Allocate` returns, one movement per `stock.Take`. `stock.ErrInsufficient` → `ErrInsufficientStock` and
  **nothing is written**. Do not re-implement ordering, expiry or tolerance.
- `BatchId` given → only that batch. If `Lot.Expired(now)` and the type is not `stock.AdjustOut` → `ErrBatchExpired`
  (writing expired stock off is exactly what `AdjustOut` is for). If its quantity is smaller → `ErrInsufficientStock`.
- Returns the list of movements.

**`RegisterEntry`** — an **inbound correction** (a count found more units than the book): `{ProductId, LocationId, BatchId,
Quantity, Note}` adds to an **existing** batch and appends a `stock.AdjustIn` movement.

**`TransferStock{ProductId, FromLocationId, ToLocationId, Quantity, Note}`** — same location → `ErrSameLocation`. Takes
`Quantity` out of the source with `stock.Allocate` (`stock.TransferOut` movements) and puts the same batches into the
destination (`stock.TransferIn`, same quantities), **atomically**.

**`ListExpiring{WithinDays}`** — stock levels with `quantity > 0` whose batch satisfies `Lot.ExpiresWithin(now,
WithinDays)` (expired ones included), as an `ExpiringStock` list `{product_id, product_name, batch_id, batch_number,
location_id, quantity, expiration_date, expired bool}` ordered by `expiration_date` ascending. `WithinDays < 0` →
`ValidationError`.

**`ListLowStock{}`** — the products that are active, have `min_quantity > 0` and whose **total quantity over all locations
and batches** is `<= min_quantity`, as `LowStock{product_id, product_name, total, min_quantity}` ordered by name. A later
`alert_manager` calls it; this module raises no alert itself.

**Atomicity and concurrency.** Each of `ReceiveStock`, `RegisterOutput`, `RegisterEntry` and `TransferStock` writes its
stock-level changes and its movements in **one database transaction**. Every stock-level update is conditional on the
`revision` read (`WHERE id = ? AND tenant_id = ? AND revision = ?`, then `revision + 1`); a lost race returns
`ErrConflict`. Copy the transaction/revision pattern of `appointment_booking` (`repository.go`,
`UpdateReservationStatusTx`; its `AGENTS.md`, «Concurrencia optimista»). Read it first.

**Multi-tenancy.** Every query and every write carries `tenant_id` in the condition; an id of another tenant is
`ErrNotFound`. No operation takes a tenant from anywhere but its args (as item_catalog does).

**Events** (`Deps.Publisher` optional; nil disables): `TopicMovementRegistered = "pharmacy.movement.registered"`
(payload: the movement), published **after** commit, once per movement. No other topic of its own.

**Cost (accounting-ready).** An output whose batch has `unit_cost > 0` consumes goods at cost. When `Deps.MoneyScope` is
non-empty (the app names the area of the business the money belongs to, e.g. `"pharmacy"`; empty disables) and
`Deps.Publisher` is non-nil, after commit publish through `money.Publish(Deps.Publisher, fact)` one fact per output
movement (`Sale`, `InternalUse`, `AdjustOut`; not entries, purchases or transfers):

```go
money.Fact{
	Kind:   money.Consumed,
	Side:   money.Outflow,
	Amount: money.Money{Amount: roundedCost, Currency: money.CLP}, // unit_cost × quantity, half away from zero; skip when it rounds to 0
	Scope:  deps.MoneyScope,
	Source: money.Ref{Type: "pharmacy_movement", ID: movementID},
	At:     createdAt,
}
```

Use `money.New` to build the `Money` and return its error if any. A publish error is **returned** to the caller after
the commit (the stock change is already stored; say so in the error text), never swallowed. Purchases are **not** a fact
of this module (a later `purchase_manager` owns them): never publish `money.Purchased` here.

**Cross-module readers** (other modules declare matching interfaces; this module's `*Module` satisfies them
structurally; do not import them): `ProductExists(tenantId, productId string) (bool, error)` — exists and active;
`ProductCard(tenantId, productId string) (name, unit, taxTreatment string, price int64, active bool, err error)` — absent or
foreign tenant ⇒ `ErrNotFound`.

## Stage 1 — scaffold

1. Delete the stub `pharmacy_manager.go`.
2. Create `model.go` (the `Definition` literals, the constants, the errors, the topic, `ModelName`), run `ormc` to
   generate `model_orm.go` (never edit it), `go mod tidy`.
3. Dependencies: the same `webtyp.com/*` set as `item_catalog/go.mod` (`fmt`, `model`, `input`, `orm`, `router`, `view`,
   `events`, `ddl`, `time`, `unixid`, `lang`, and for `ui/` and `web/`: `dom`, `layout`, `svg`, `components`, `storage`)
   **plus** `webtyp.com/stock` v0.1.0 and `webtyp.com/money` v0.1.0 (and `webtyp.com/events` at the version
   `go get webtyp.com/events@latest` resolves, which must be v0.1.0 or higher). Use the versions item_catalog's `go.mod`
   pins for the rest; **no local `replace`**.
4. `migrate/migrate.go`: `Migrate(conn ddl.Execer, c ddl.Compiler) error` creating the five tables in FK order
   (`Product`, `Location`, `Batch`, `StockLevel`, `Movement`), like item_catalog's.

## Stage 2 — the module

`module.go`: `Deps{IDs model.IDGenerator (required), Publisher events.Publisher (optional), MoneyScope string (optional)}`,
`Module`, `New` (error text `"pharmacy_manager: Deps.IDs is required"`), the service methods named in the operations table,
the two readers. `mcp.go`/`ops.go`: `ModelName()`, `MountOperations`, the handlers and `writeError`/`statusOf`,
exactly as item_catalog's. `interfaces.go` is **not** needed (this module defines no port).

Time: use `webtyp.com/time` for "now" (Unix seconds). Do **not** add a clock parameter, option or seam for tests: tests
use expiry dates far in the future and in the past.

## Stage 3 — view

`view.go`: `Item()` for `Product` (`Label` = name, `Description` = sku) and `Location` (`Label` = name);
`NewProductView(caller) view.Presenter`, `NewLocationView(caller) view.Presenter`, built like item_catalog's `NewView`
(`view.NewCallerLister` + `view.New`), with titles "Products" and "Locations" as named constants.

## Stage 4 — `ui/`, `seed/`, `web/`

- `ui/module.go`: `ID = "pharmacy"`, `Label = "Farmacia"`, icons `IconPharmacy = svg.Icon("pharmacy")`,
  `IconProducts = svg.Icon("pharmacy_products")`, `IconLocations = svg.Icon("pharmacy_locations")`.
- `ui/svg.go` (`//go:build !wasm`): one sprite entry per icon, all three with the same simple box geometry
  (`viewBox "0 0 16 16"`, one path): `M2 4l6-3 6 3v8l-6 3-6-3V4zm6-1.7L4.2 4.2 8 6.1l3.8-1.9L8 2.3zM3 5.4v6l4.5 2.2V7.6L3 5.4zm10 0L8.5 7.6v6L13 11.4v-6z`.
  Do not draw other artwork.
- `ui/browser.go`: `Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error)`
  returns `platformd.NewGroupUIModule(ID, Label, IconPharmacy, children)` with two children built as in
  `item_catalog/ui/browser.go`: `ID+".products"` ("Productos", a `crudview` over `NewProductView`) and
  `ID+".locations"` ("Ubicaciones", a `crudview` over `NewLocationView`). `NewRecord` for a product: `Unit: UnitPiece, TaxTreatment: TaxTaxed, IsActive: true`;
  for a location: `IsActive: true`. **No screen for stock, batches or movements in this version** (a later plan designs the
  entry/exit forms); say so in `docs/ARCHITECTURE.md`.
- `seed/seed.go`: `Load(m *Module, tenantID string) (Data, error)` through the module's own methods: two locations ("Bodega" and "Mostrador", the second `is_sales_point`), three products (one for sale with a price of 2990
and `TaxTaxed`, one for sale `TaxExempt`, one not for sale), one batch received for each with a future expiry and a unit
cost on two of them.
- `web/client.go` (`//go:build wasm`): the runnable in-browser demo, copy of item_catalog's with `pharmacymanager`
  (`storage/mem`, `router/loopback`, the seed, `Browser`).
- All user-visible text in `ui/` and `view.go` is `lang.Text` with English keys; run `langc sync` and add the Spanish
  translations in `lang.json` (Productos, Ubicaciones, Farmacia, Nombre, Unidad, Tratamiento tributario, Afecto, Exento, Para la venta, Precio, Controlado, Código de barras, N.º de registro sanitario, Cantidad mínima, Activo, Punto de venta, and every error message that reaches the person). The Spanish is
  gender-neutral (skill **translations**).

## Stage 5 — tests (`tests/`, external package `pharmacymanager_test` unless noted, `orm.New(mem.New())`, no mocks of the module)

Behaviour and rules only, never the look. Group them in files by feature; do not duplicate a rule in two files. Every
test file that imports `webtyp.com/stock/stocktest` carries `//go:build !wasm`.

1. **Conformance of the shared rules:** a small `ledger_test.go` adapter implements `stocktest.Ledger` on top of this
   module (it creates the product and the location on first use and maps the suite's ids to them) and calls
   `stocktest.Run(t, newLedger)`. This is the proof that this module obeys the same stock rules as its sibling. Do not
   copy the suite's cases into other tests.
2. **Products:** create/get/update/deactivate/delete; `ErrSkuExists` (per tenant, not across tenants); `ErrSaleNeedsPrice`; `ErrTaxTreatmentRequired`; `price` stored as 0 when not for sale;
   delete with a batch → `ErrProductInUse`; foreign tenant → `ErrNotFound`.
3. **Locations:** unique name, delete in use.
4. **Batches and receiving:** a batch without expiry → `ErrExpiryRequired`; a blank batch number → `ValidationError`; receiving twice into the same batch adds up; different expiry →
   `ErrBatchExpiryMismatch`; different non-zero cost → `ErrBatchCostMismatch`; cost 0 into an existing batch keeps it.
5. **Output:** the `Sale`, `InternalUse` and `AdjustOut` types work and others are `ValidationError`; a failed output leaves stock and
   movements unchanged; forced `BatchId` expired → `ErrBatchExpired` unless `AdjustOut`; a `Sale` of a product not for sale → `ErrNotForSale`; an expired batch is skipped by FEFO and refused when forced.
6. **Transfer:** moves the same batches, `ErrSameLocation`, insufficient stock leaves both locations unchanged.
7. **Invariants:** quantity stored is always > 0; stock never negative after any sequence; the sum of `list_stock` equals
   the sum of in-movements minus out-movements for a product (a property test over a short scripted sequence).
8. **Concurrency:** a stale `revision` on a stock-level update returns `ErrConflict`. The public API cannot force a stale
   revision, so this one test is a **root-level test** (`conflict_internal_test.go`, package `pharmacymanager`) whose first line is
   `// Root-level test (justified): exercises the unexported stock-level update — the public API cannot force a stale revision.`
9. **`ListExpiring` and `ListLowStock`:** expiring includes expired and soon-to-expire, excludes far-future and empty
   rows; negative days → error; low stock includes the product at exactly `min_quantity`, excludes `min_quantity == 0` and
   inactive ones, and sums across locations and batches.
10. **Events and cost:** one `pharmacy.movement.registered` per movement, none when `Publisher` is nil, none on a failed
    operation; with `MoneyScope` set and a batch cost, each output publishes one `money.Consumed` fact with the right
    amount (check a fractional quantity rounds half away from zero), a purchase publishes none, no `MoneyScope` →
    none, cost 0 → none. Use `events/mock`'s broker and `money.Subscribe` with a recording `money.Sink`.
11. **Conformance of the module:** it satisfies `router.OperationModule`; every op is mounted with `Requires` and
    `Accepts` (copy item_catalog's `conformance_test.go`).
12. **View test** (`tests/pharmacy_view_test.go` + wasm variant as item_catalog's): the group has the two children; the
    product form renders its fields with their labels.

## Stage 6 — docs

`README.md` (quick start, Ops table, readers, the cost fact, key files), `docs/ARCHITECTURE.md` (scope, entities, rules
above, the "no stock screen yet" note, the relation to `webtyp/stock` and `webtyp/money`, composition-root example that
shows `Deps{IDs, Publisher, MoneyScope}`), `docs/diagrams/database.md` (Mermaid ERD of the five tables). Update the
`AGENTS.md` domain notes only if this plan contradicts them. Delete nothing else. `docs/PLAN.md` itself is deleted by
the workflow, never by you.

## Delivery check

```
gotest                                                  # vet, race, tests and wasm green
grep -rn "errors\." --include=*.go . | grep -v _test    # → empty (no stdlib errors)
grep -rn "Purchased" --include=*.go . | grep -v _test   # → empty (purchases are not a fact of this module)
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
