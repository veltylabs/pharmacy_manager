# Architecture

## Scope

`pharmacy_manager` holds the clinic's pharmacy, open to the public: medicines, the batches they come in, where they are
kept and every movement of stock. Clinical supplies used in consultations are a different module (`supply_manager`) with
other managers, permissions and rules. Both use the rules of `webtyp.com/stock` and its conformance suite; they share no
table and no code, and neither imports the other.

There is no stock screen in this version: the screen group "Farmacia" edits products and locations only.

## Entities

| Table | What it is |
|---|---|
| `pharmacy_product` | a medicine or article: SKU, name, unit, tax treatment, sale flag and price (whole pesos, tax included), minimum quantity |
| `pharmacy_location` | a place stock is kept (storeroom, counter); `is_sales_point` marks where sales happen |
| `pharmacy_batch` | one lot of one product: batch number, mandatory expiry (Unix seconds UTC), optional unit cost |
| `pharmacy_stock_level` | how much of one batch is in one location, with a revision for concurrent updates |
| `pharmacy_movement` | one append-only entry of the ledger |

Diagram: [diagrams/database.md](diagrams/database.md).

## Rules

- **All stock is held per batch.** The rules of stock (first expiry first out, expired lots, never negative, movement directions) are those of `webtyp.com/stock`; nothing in this module repeats them.
- **A movement is append-only**: there is no update and no delete; a mistake is corrected by a new movement. Its `quantity` is always greater than zero; the direction comes from the movement type.
- **Every stock change is one transaction** with an optimistic revision check on the stock level (`ErrConflict` when the row changed).
- **Cost**: a batch has an optional `unit_cost` (whole pesos per unit, 0 = unknown). With `Deps.Publisher` and `Deps.MoneyScope` set, each output movement whose batch has a cost publishes one `money.Consumed` fact (cost × quantity, rounded half away from zero). A purchase is never a fact of this module.
- **Alerts**: raises an alert when a product crosses below its minimum, and another when the application runs the daily `ReviewExpiring` review of batches.
- **Events**: one `pharmacy.movement.registered` per movement, after the commit.
- **Tenancy**: every query of a stock operation filters by `tenant_id`; the operations the generic view drives resolve the row by id and use the tenant the row stores.
- Units: `PIECE`, `BOX`, `TABLET`, `ML`, `GRAM`. Tax treatment: `TAXED` or `EXEMPT` (the rate is the law's and belongs to the accounting configuration, not here). Prices are whole pesos, tax included.
- A batch **must have an expiry date** (`ErrExpiryRequired`). A product for sale needs a price greater than zero (`ErrSaleNeedsPrice`); a product not for sale is stored with price 0. A `stock.Sale` output of a product that is not for sale is `ErrNotForSale`. An expired batch is never sold or used (`ErrBatchExpired`); only a write-off (`stock.AdjustOut`) may take expired stock.
- `controlled` is only a flag in this version: no logic depends on it; the regulated control book of narcotics and psychotropics stays outside the software.
- Outputs accepted: `stock.Sale`, `stock.InternalUse`, `stock.AdjustOut`. `min_quantity` (0 = no alert) feeds `ListLowStock`, which a later alert module reads.
- **Time units**: expiry dates are Unix **seconds**; `webtyp.com/time.Now()` returns Unix **nanoseconds**. The module
  compares an expiry only with `nowSeconds()`; `created_at` and `updated_at` keep the nanoseconds of `time.Now()`.

## Dependencies

- `webtyp.com/stock` — first expiry first out (`Allocate`), never-negative arithmetic (`Apply`), movement directions and
  expiry predicates. No stock rule is written in this module.
- `webtyp.com/money` — the `money.Consumed` cost facts, published through `money.Publish` on the module's `events.Publisher`.

## Composition root

```go
pm, err := pharmacymanager.New(db, pharmacymanager.Deps{
	IDs:        ids,          // required
	Publisher:  broker,       // optional: nil disables the events and the cost facts
	MoneyScope: "pharmacy",   // optional: "" disables the cost facts
})
registry.Register(pm)         // router.OperationModule
screens := ui.Screens(caller)
```

El esquema se concilia con `migrate.Migrate` usando `ddl.Sync` desde el paso de migración de la app (flujo: https://github.com/webtyp/ddl/blob/main/docs/diagrams/SYNC_FLOW.md).

## Operations the generic view drives

`upsert_*`, `update_*` and `delete_*` carry exactly what `webtyp.com/view` sends (`{records}`, `{ids, fields, record}`,
`{ids}`) and therefore no `tenant_id`: they resolve each row by its id and work with the tenant the row itself stores.
Every other operation takes `tenant_id` and filters by it.
