# pharmacy_manager
<img src="docs/img/badges.svg">

## ¿Por qué existe?
`pharmacy_manager` gestiona la farmacia clínica abierta al público: medicamentos, los lotes en los que vienen con su caducidad, dónde se mantienen y cada movimiento de stock.
**Sin esto vs. Con esto**: Sin esto la clínica necesitaría herramientas de inventario externas desvinculadas. Con esto se integran las reglas complejas de [stock](https://github.com/webtyp/stock) y operaciones estandarizadas a través de [crud](https://github.com/webtyp/crud), orquestados como describe [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Antes de comenzar
Requiere Go configurado y librerías privadas `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`.
Instanciar con base de datos `*orm.DB` que contenga las tablas creadas por [migrate/migrate.go](migrate/migrate.go).
Dependencias de configuración `Deps`: `Deps.IDs` y `Deps.TenantID` son obligatorias. Opcionales: `Deps.Publisher`, `Deps.MoneyScope`, `Deps.AlertArea`.

## Guía paso a paso

Paso 1:
```go
package main

import (
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
)

func main() {
    db := orm.New(mem.New())
	m, _ := pharmacymanager.New(db, pharmacymanager.Deps{TenantID: "demo", IDs: myIDGen})
	p, _ := m.CreateProduct(pharmacymanager.PharmacyProduct{Sku: "P1", Name: "Med"})
	l, _ := m.CreateLocation(pharmacymanager.PharmacyLocation{Name: "Almacen"})
	m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ProductId: p.Id, LocationId: l.Id, Quantity: 10, BatchNumber: "L1"})
    // fmt.Print(m.ListStock(...))
}
```

Paso 2: También expuesto a través de operaciones asertivas (ver [tests/rules_test.go](tests/rules_test.go)).
Paso 3: Ver en la interfaz con la demo de webtyp, ver [seed/demo.go](seed/demo.go).

## Mapa de documentación
| Orden | Documento | Qué pregunta responde y cuándo leerlo |
|---|---|---|
| 1 | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Visión general del modelo de datos y diseño. |
| 2 | [docs/diagrams/STOCK_OPERATION.md](docs/diagrams/STOCK_OPERATION.md) | ¿Qué flujo interno sigue un movimiento de stock? |
| 3 | [docs/TESTS.md](docs/TESTS.md) | Reglas de dominio implementadas. |

Antes / Después de las operaciones CRUD: Todo movimiento expuesto en `pharmacy_operation`.
Permisos:
- `pharmacy_product`, `pharmacy_location`: crear, leer, actualizar, borrar.
- `pharmacy_operation`: crear, leer.
- `pharmacy_batch`, `pharmacy_stock_level`, `pharmacy_movement`, `expiring_stock`, `low_stock`: leer.
