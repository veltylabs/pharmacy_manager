# pharmacy_manager
<img src="docs/img/badges.svg">

The clinic's pharmacy, open to the public: the medicines, the batches they come in (every batch has a mandatory expiry
date), the locations they are kept in and the append-only ledger of every stock movement. The stock rules
(first expiry first out, expired batches unused, never negative) come from the library `webtyp.com/stock`.

It is not a sales, billing or accounting module and knows nothing about services, specialties or patients. A sale is an
output movement (`stock.Sale`) that carries the id of the document that caused it; the document belongs to another module.

## Quick start

```go
import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/stock"
)

pm, err := pharmacymanager.New(db, pharmacymanager.Deps{IDs: ids, Publisher: broker, MoneyScope: "pharmacy"})
if err != nil {
	return err
}
counter, _ := pm.CreateLocation(pharmacymanager.PharmacyLocation{TenantId: tenant, Name: "Mostrador", IsSalesPoint: true, IsActive: true})
para, _ := pm.CreateProduct(pharmacymanager.PharmacyProduct{
	TenantId: tenant, Sku: "PARA500", Name: "Paracetamol 500 mg", Unit: pharmacymanager.UnitBox,
	TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 2990, IsActive: true,
})
// Expiry dates are Unix SECONDS; webtyp.com/time.Now() counts nanoseconds.
expiry := time.Now()/1000000000 + 365*stock.SecondsPerDay
pm.ReceiveStock(pharmacymanager.ReceiveStockArgs{
	TenantId: tenant, ProductId: para.Id, LocationId: counter.Id, BatchNumber: "L-2026-01",
	ExpirationDate: expiry, UnitCost: 300, Quantity: 40,
})
pm.RegisterOutput(pharmacymanager.RegisterOutputArgs{
	TenantId: tenant, ProductId: para.Id, LocationId: counter.Id, Quantity: 2,
	MovementType: string(stock.Sale), SourceDocumentType: "sale", SourceDocumentId: saleID,
})
```

## Operations

Every operation is called `pharmacy.<name>` and declares its RBAC resource and action in `MountOperations`.

| Operation | Constant | Resource | Action |
|---|---|---|---|
| `list_products` | `OpListProducts` | `pharmacy_product` | read |
| `get_product` | `OpGetProduct` | `pharmacy_product` | read |
| `create_product` | `OpCreateProduct` | `pharmacy_product` | create |
| `update_product` | `OpUpdateProduct` | `pharmacy_product` | update |
| `upsert_product` | `OpUpsertProduct` | `pharmacy_product` | create + update |
| `deactivate_product` | `OpDeactivateProduct` | `pharmacy_product` | update |
| `delete_product` | `OpDeleteProduct` | `pharmacy_product` | delete |
| `list_locations` | `OpListLocations` | `pharmacy_location` | read |
| `upsert_location` | `OpUpsertLocation` | `pharmacy_location` | create + update |
| `update_location` | `OpUpdateLocation` | `pharmacy_location` | update |
| `delete_location` | `OpDeleteLocation` | `pharmacy_location` | delete |
| `list_batches` | `OpListBatches` | `pharmacy_batch` | read |
| `receive_stock` | `OpReceiveStock` | `pharmacy_movement` | create |
| `register_entry` | `OpRegisterEntry` | `pharmacy_movement` | create |
| `register_output` | `OpRegisterOutput` | `pharmacy_movement` | create |
| `transfer_stock` | `OpTransferStock` | `pharmacy_movement` | create |
| `list_stock` | `OpListStock` | `pharmacy_stock_level` | read |
| `list_movements` | `OpListMovements` | `pharmacy_movement` | read |
| `list_expiring` | `OpListExpiring` | `pharmacy_stock_level` | read |
| `list_low_stock` | `OpListLowStock` | `pharmacy_stock_level` | read |

## Readers for other modules

`ProductExists(tenantId, productId) (bool, error)` — the product exists in the tenant and is active.
`ProductCard(tenantId, productId) (name, unit, taxTreatment string, price int64, active bool, err error)` — what another
module snapshots about a product; an absent or foreign product is `ErrNotFound`. A module that needs them declares the
method in a narrow interface of its own; `*Module` satisfies it without importing that module.

## Alerts

With `Deps.AlertArea` and `Deps.Publisher` set, the module raises `notify.Alert` events for two rules:
- **Below minimum:** After an output, if a product's total stock crosses from above to at or below its `min_quantity`, one alert for that product is raised (`pharmacy.below_minimum`).
- **Expiring:** The application calls `ReviewExpiring(tenantId, withinDays)` once a day to review expiring batches. It raises one summary alert (`pharmacy.expiring`) when batches are expired or expire within `withinDays`.
The catalog of alerts is returned by `AlertKinds(area string)`.

## Cost facts

With `Deps.Publisher` and `Deps.MoneyScope` set, every output movement whose batch has a `unit_cost` publishes one
`money.Consumed` fact through `money.Publish`: cost × quantity in whole pesos, rounded half away from zero (333 × 2.5 =
833). A batch without cost (0) publishes nothing. A purchase is never a fact of this module. Every movement also
publishes `pharmacy.movement.registered` with the movement as payload, after the transaction commits.

## Key files

| File | What it is |
|---|---|
| `model.go`, `model_orm.go` | the five tables and every argument model; `model_orm.go` is generated by `ormc`, never edit it |
| `errors.go` | every error constant (all messages start `pharmacy: `), `statusOf`, `writeError` |
| `module.go` | `Deps`, `Module`, `New`, the four stock operations `ReceiveStock`, `RegisterEntry`, `RegisterOutput`, `TransferStock`, cost facts and events |
| `crud.go` | product and location update, upsert, patch, deactivate and delete |
| `queries.go`, `reports.go` | the lists, `ListExpiring`, `ListLowStock` and the readers |
| `ops.go` | operation names, the wire shapes the generic view sends, `MountOperations` and every handler |
| `view.go` | `NewProductView`, `NewLocationView` |
| `migrate/migrate.go` | creates the five tables in foreign-key order |
| `ui/`, `seed/`, `web/` | the screen group "Farmacia", the demo data, the in-browser demo |
| `lang.json` | Spanish of the labels and options |
| `conflict_internal_test.go` | the stale-revision test (root level, justified) |
| `tests/` | the tests, including the shared `stocktest` conformance suite (`ledger_test.go`) |

More: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/diagrams/database.md](docs/diagrams/database.md).
