← [Stage 1](PLAN_STAGE_1_TOOLS_DEFINITION.md) | Next → [Stage 3](PLAN_STAGE_3_STOCK_TX.md)

# Stage 2 — `Module.store`, `storeFor`, row helpers, translation table

## 2.1 `module.go`, header only (the methods are Stage 5)

- `type Module struct`: add `store *orm.Store` after `db`.
- `Deps.TenantID` becomes **required** (its comment: `// required: the tenant of the installation; the module's store is sealed to it`). In `New`, before the return: `if deps.TenantID == "" { return nil, fmt.Err(msgNeedTenant) }`, with the new constant `msgNeedTenant = "pharmacy: Deps.TenantID is required"` next to `msgNeedIDs`; in the returned literal add `store: orm.TenantStore(db, deps.TenantID, deps.IDs),`.
- Delete `func (m *Module) ModelName() string { return ModelName }` (generated in Stage 4). Everything else of `module.go` stays untouched in this stage.

## 2.2 `store.go` (new), written exactly

```go
package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/orm"
)

// StorePharmacyProduct es el repositorio de los productos.
//
// Ejemplo: store := m.StorePharmacyProduct()
func (m *Module) StorePharmacyProduct() crud.Store { return m.store }

// StorePharmacyLocation es el repositorio de las ubicaciones.
//
// Ejemplo: store := m.StorePharmacyLocation()
func (m *Module) StorePharmacyLocation() crud.Store { return m.store }

// StorePharmacyBatch es el repositorio de los lotes.
//
// Ejemplo: store := m.StorePharmacyBatch()
func (m *Module) StorePharmacyBatch() crud.Store { return m.store }

// StorePharmacyStockLevel es el repositorio de los niveles de stock.
//
// Ejemplo: store := m.StorePharmacyStockLevel()
func (m *Module) StorePharmacyStockLevel() crud.Store { return m.store }

// StorePharmacyMovement es el repositorio de los movimientos.
//
// Ejemplo: store := m.StorePharmacyMovement()
func (m *Module) StorePharmacyMovement() crud.Store { return m.store }

// StorePharmacyOperation es el repositorio de las operaciones de stock.
//
// Ejemplo: store := m.StorePharmacyOperation()
func (m *Module) StorePharmacyOperation() crud.Store { return m.store }

// storeFor es el repositorio de un tenant para los métodos que reciben el tenant como argumento: el del módulo es
// m.store; otro tenant tiene su propio repositorio sellado, que no ve las filas de los demás. Un tenant vacío es un error:
// un repositorio sin tenant vería todas las filas.
//
// Ejemplo: store, err := m.storeFor("tenant-1")
func (m *Module) storeFor(tenantId string) (*orm.Store, error) {
	if tenantId == "" {
		return nil, ErrTenantRequired
	}
	if tenantId == m.tenantID {
		return m.store, nil
	}
	return orm.TenantStore(m.db, tenantId, m.ids), nil
}
```
If `definitionc` names the `Store…` methods differently (it reads the entity names), keep the names it asks for in its error message.

## 2.3 `rows.go` (new), written exactly

```go
package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// firstRow responde la primera fila que cumple conds y si existe alguna.
//
// Ejemplo: row, found, err := firstRow(tx, func() model.Model { return &PharmacyProduct{} }, storage.Eq(PharmacyProduct_.Sku, "p1"))
func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

// allRows responde todas las filas que cumplen conds, paginando por id porque una lectura trae como máximo crud.MaxPage filas.
//
// Ejemplo: rows, err := allRows(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, "p1"))
func allRows(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) ([]model.Model, error) {
	var rows []model.Model
	after := ""
	for {
		var page []model.Model
		err := tx.ReadAll(newRow, crud.Query{Conditions: conds, After: after, Limit: crud.MaxPage},
			func(r model.Model) { page = append(page, r) })
		if err != nil {
			return nil, err
		}
		rows = append(rows, page...)
		if len(page) < crud.MaxPage {
			return rows, nil
		}
		after = keyOf(page[len(page)-1])
	}
}

// keyOf responde el id de una fila de las tablas de este módulo.
func keyOf(r model.Model) string {
	switch row := r.(type) {
	case *PharmacyProduct:
		return row.Id
	case *PharmacyLocation:
		return row.Id
	case *PharmacyBatch:
		return row.Id
	case *PharmacyStockLevel:
		return row.Id
	case *PharmacyMovement:
		return row.Id
	case *PharmacyOperation:
		return row.Id
	}
	return ""
}

// notFound cambia el error «no existe» de un repositorio por `missing` y deja pasar cualquier otro error.
//
// Ejemplo: err = notFound(err, ErrProductNotFound)
func notFound(err error, missing error) error {
	if model.KindOf(err) == model.ErrorNotFound {
		return missing
	}
	return err
}

// containsString dice si list contiene value.
//
// Ejemplo: ok := containsString([]string{"a", "b"}, "b")
func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
```
(`orm.IsNotFound` of the old code does not apply to `crud.Tx`: use `notFound`.)

## 2.4 Translation table for Stages 3–5 (old `m.db` / `tx *orm.DB` call → new)

| Old | New |
|---|---|
| `db.Query(&x).Where(X_.Id).Eq(id).Where(X_.TenantId).Eq(t)` + `ReadOneX` + `orm.IsNotFound` | `err := tx.ReadOne(&x, id)`, then `notFound(err, ErrXNotFound)` (the tenant is the store's) |
| `ReadOneX(db.Query(&x).Where(X_.A).Eq(v).Where(...tenant...), &x)` | `row, found, err := firstRow(tx, func() model.Model { return &X{} }, storage.Eq(X_.A, v))`, then `*row.(*X)` |
| `ReadAllX(db.Query(&proxy).Where(X_.A).Eq(v)...)` | `rows, err := allRows(tx, func() model.Model { return &X{} }, storage.Eq(X_.A, v))`, then `*rows[i].(*X)` |
| `tx.Create(&x)` | `tx.Create(&x)` (id from `m.ids.NewID()` as today) |
| `tx.Update(got, orm.Eq(...), orm.Eq(...))` | `tx.UpdateFields(&row, id, cols)` with the columns the old code changed |
| `m.db.Delete(&p, orm.Eq(...), orm.Eq(...))` | `tx.Delete(&p, id)` |
| `m.db.Tx(func(tx *orm.DB) error {…})` | `store.Atomic(func(tx crud.Tx) error {…})` with `store` from `m.storeFor(tenantId)` (kept methods) or `m.store` (id-only methods: `PatchProducts`, `DeleteProducts`, `PatchLocations`, `DeleteLocations`) |

Commit and push.
