# pharmacy_manager
<img src="docs/img/badges.svg">

La farmacia de la clínica, abierta al público: los medicamentos, los lotes en los que vienen (cada lote tiene una fecha de caducidad obligatoria), las ubicaciones en las que se guardan y el libro mayor (solo de anexos) de cada movimiento de existencias. Las reglas de existencias (primero en caducar primero en salir, lotes caducados no utilizados, nunca negativo) provienen de la biblioteca `webtyp.com/stock`.

No es un módulo de ventas, facturación o contabilidad y no sabe nada sobre servicios, especialidades o pacientes. Una venta es un movimiento de salida (`stock.Sale`) que lleva el id del documento que la provocó; el documento pertenece a otro módulo.

## Inicio rápido

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
// Las fechas de caducidad son SEGUNDOS Unix; webtyp.com/time.Now() cuenta nanosegundos.
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

## Operaciones

Cada operación se llama `pharmacy.<nombre>` y declara su recurso y acción RBAC en `MountOperations`.

| Operación | Constante | Recurso | Acción |
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

## Lectores para otros módulos

`ProductExists(tenantId, productId) (bool, error)` — el producto existe en el inquilino y está activo.
`ProductCard(tenantId, productId) (name, unit, taxTreatment string, price int64, active bool, err error)` — lo que otro
módulo captura sobre un producto; un producto ausente o extranjero es `ErrNotFound`. Un módulo que los necesita declara el
método en una interfaz estrecha propia; `*Module` la satisface sin importar ese módulo.

## Alertas

Con `Deps.AlertArea` y `Deps.Publisher` establecidos, el módulo lanza eventos `notify.Alert` para dos reglas:
- **Por debajo del mínimo:** Después de una salida, si el stock total de un producto pasa de estar por encima a estar en o por debajo de su `min_quantity`, se lanza una alerta para ese producto (`pharmacy.below_minimum`).
- **A punto de caducar:** La aplicación llama a `ReviewExpiring(tenantId, withinDays)` una vez al día para revisar los lotes a punto de caducar. Lanza una alerta de resumen (`pharmacy.expiring`) cuando los lotes han caducado o caducan dentro de `withinDays`.
El catálogo de alertas es devuelto por `AlertKinds(area string)`.

## Datos de costos

Con `Deps.Publisher` y `Deps.MoneyScope` establecidos, cada movimiento de salida cuyo lote tiene un `unit_cost` publica un
hecho `money.Consumed` a través de `money.Publish`: costo × cantidad en pesos enteros, redondeado a la mitad desde cero (333 × 2,5 =
833). Un lote sin costo (0) no publica nada. Una compra nunca es un hecho de este módulo. Cada movimiento también
publica `pharmacy.movement.registered` con el movimiento como carga útil, después de que se confirma la transacción.

## Archivos clave

| Archivo | Qué es |
|---|---|
| `model.go`, `model_orm.go` | las cinco tablas y cada modelo de argumento; `model_orm.go` es generado por `ormc`, nunca lo edite |
| `errors.go` | cada constante de error (todos los mensajes comienzan con `pharmacy: `), `statusOf`, `writeError` |
| `module.go` | `Deps`, `Module`, `New`, las cuatro operaciones de existencias `ReceiveStock`, `RegisterEntry`, `RegisterOutput`, `TransferStock`, datos de costos y eventos |
| `crud.go` | actualización, inserción/actualización (upsert), parche, desactivación y eliminación de productos y ubicaciones |
| `queries.go`, `reports.go` | las listas, `ListExpiring`, `ListLowStock` y los lectores |
| `ops.go` | nombres de operaciones, las formas de cableado que envía la vista genérica, `MountOperations` y cada manejador |
| `view.go` | `NewProductView`, `NewLocationView` |
| `migrate/migrate.go` | crea las cinco tablas en orden de clave externa |
| `ui/`, `seed/`, `web/` | las pantallas, los datos de demostración, la demostración en el navegador |
| `lang.json` | Español de las etiquetas y opciones |
| `conflict_internal_test.go` | la prueba de revisión obsoleta (nivel raíz, justificado) |
| `tests/` | las pruebas, incluida la suite de conformidad compartida `stocktest` (`ledger_test.go`) |

Más: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/diagrams/database.md](docs/diagrams/database.md).
