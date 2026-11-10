← [Stage 3](PLAN_STAGE_3_STOCK_TX.md) | Next → [Stage 5](PLAN_STAGE_5_METHODS.md)

# Stage 4 — generate, shared checks, rule files

## 4.1 Generate

1. Delete `model_orm.go` and `definition_crud.go` if they exist (not `ops.go`, `crud.go`, `queries.go`, `reports.go`: Stage 5 rewrites them). Run `definitionc` in the repo root. It writes `definition_gen.go` (types, `<Entity>List`, column constants `PharmacyProduct_`, `PharmacyOperation_`…, `ModelName`, `ModelName()`, `Definition()`, `MountOperations()`) and one skeleton per action declared `AddMyBusinessRule`. Then repeat the `go get` line of Stage 1 and `go mod tidy`.
2. Expected skeletons: `rule_pharmacy_product_{create,update,delete}.go`, `rule_pharmacy_location_{create,update,delete}.go`, `rule_pharmacy_operation_create.go`, `rule_expiring_stock_read.go`, `rule_low_stock_read.go`. **Keep the signatures `definitionc` wrote**; replace only the bodies with the ones below (if a file name differs, keep the generated one).

## 4.2 `checks.go` (new) — what a rule and its kept method share, written exactly

`checkProduct` (unit, tax, price) stays in `module.go`: the rule and `CreateProduct` both call it.

```go
package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// checkSkuFree rechaza un SKU que ya tiene otro producto; lo comparten la regla de crear/actualizar producto y los métodos.
// Ejemplo: err := checkSkuFree(tx, "PARA500")   // ErrSkuExists si ya existe
func checkSkuFree(tx crud.Tx, sku string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyProduct{} }, storage.Eq(PharmacyProduct_.Sku, sku))
	if err != nil {
		return err
	}
	if found {
		return ErrSkuExists
	}
	return nil
}

// checkLocationNameFree rechaza un nombre que ya tiene otra ubicación.
// Ejemplo: err := checkLocationNameFree(tx, "Bodega")   // ErrLocationNameExists si ya existe
func checkLocationNameFree(tx crud.Tx, name string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyLocation{} }, storage.Eq(PharmacyLocation_.Name, name))
	if err != nil {
		return err
	}
	if found {
		return ErrLocationNameExists
	}
	return nil
}

// checkProductDeletable rechaza borrar un producto que tiene lotes: el libro mayor conserva su historia.
// Ejemplo: err := checkProductDeletable(tx, "p1")   // ErrProductInUse si tiene lotes
func checkProductDeletable(tx crud.Tx, productId string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, productId))
	if err != nil {
		return err
	}
	if found {
		return ErrProductInUse
	}
	return nil
}

// checkLocationDeletable rechaza borrar una ubicación que alguna fila de nivel de stock referencia.
// Ejemplo: err := checkLocationDeletable(tx, "l1")   // ErrLocationInUse si tiene stock
func checkLocationDeletable(tx crud.Tx, locationId string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyStockLevel{} }, storage.Eq(PharmacyStockLevel_.LocationId, locationId))
	if err != nil {
		return err
	}
	if found {
		return ErrLocationInUse
	}
	return nil
}

// applyProductFields copia a dst solo los campos de fields tomando sus valores de src; un campo desconocido es un error de
// campo. Lo comparten la regla de actualizar producto y PatchProducts.
// Ejemplo: err := applyProductFields(&current, &rec, []string{"name", "price"})
func applyProductFields(dst, src *PharmacyProduct, fields []string) error {
	for _, f := range fields {
		switch f {
		case PharmacyProduct_.Sku:
			dst.Sku = src.Sku
		case PharmacyProduct_.Name:
			dst.Name = src.Name
		case PharmacyProduct_.Unit:
			dst.Unit = src.Unit
		case PharmacyProduct_.TaxTreatment:
			dst.TaxTreatment = src.TaxTreatment
		case PharmacyProduct_.ForSale:
			dst.ForSale = src.ForSale
		case PharmacyProduct_.Price:
			dst.Price = src.Price
		case PharmacyProduct_.Controlled:
			dst.Controlled = src.Controlled
		case PharmacyProduct_.Barcode:
			dst.Barcode = src.Barcode
		case PharmacyProduct_.RegistryNumber:
			dst.RegistryNumber = src.RegistryNumber
		case PharmacyProduct_.MinQuantity:
			dst.MinQuantity = src.MinQuantity
		case PharmacyProduct_.IsActive:
			dst.IsActive = src.IsActive
		default:
			return &model.FieldError{Field: f, Err: ErrUnknownField}
		}
	}
	return nil
}

// applyLocationFields es applyProductFields de las ubicaciones.
// Ejemplo: err := applyLocationFields(&current, &rec, []string{"name"})
func applyLocationFields(dst, src *PharmacyLocation, fields []string) error {
	for _, f := range fields {
		switch f {
		case PharmacyLocation_.Name:
			dst.Name = src.Name
		case PharmacyLocation_.IsSalesPoint:
			dst.IsSalesPoint = src.IsSalesPoint
		case PharmacyLocation_.IsActive:
			dst.IsActive = src.IsActive
		default:
			return &model.FieldError{Field: f, Err: ErrUnknownField}
		}
	}
	return nil
}
```
(The old `PatchProducts` / `PatchLocations` used string literals `"sku"`…; the generated constants hold the same strings. `Controlled`, `Barcode`… are the generated Go names of the columns; if one differs, use the generated name.)

## 4.3 The rule files (bodies; keep each generated signature and write the Spanish comment on top)

`rule_pharmacy_product_create.go`:

```go
// Create es la regla de negocio de crear pharmacy_product: unidad, tratamiento y precio válidos, y un SKU que nadie usa.
func (p *PharmacyProduct) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkProduct(p); err != nil {
		return 0, err
	}
	if err := checkSkuFree(tx, p.Sku); err != nil {
		return 0, err
	}
	p.UpdatedAt = time.NowNano()
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_product_update.go` — the receiver carries only the sent `fields`; the rule merges them into the stored row, checks the merged product and writes the derived columns (`price` is zeroed when the product is not for sale; `updated_at` always):

```go
// Update es la regla de negocio de actualizar pharmacy_product: old es la fila guardada y el receptor trae solo los campos
// enviados (fields). Revisa el producto resultante y, si cambió el SKU, que nadie más lo use.
func (next *PharmacyProduct) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyProduct, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	if err := applyProductFields(&merged, next, fields); err != nil {
		return 0, err
	}
	if err := checkProduct(&merged); err != nil {
		return 0, err
	}
	if merged.Sku != old.Sku {
		if err := checkSkuFree(tx, merged.Sku); err != nil {
			return 0, err
		}
	}
	merged.UpdatedAt = time.NowNano()
	next.Price = merged.Price // crud escribe next con las columnas de fields: si price viaja, debe ser el normalizado
	cols := []string{PharmacyProduct_.Price, PharmacyProduct_.UpdatedAt}
	if err := tx.UpdateFields(&merged, old.Id, cols); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_product_delete.go`:

```go
// Delete es la regla de negocio de borrar pharmacy_product: un producto con lotes no se borra. El receptor es la fila guardada.
func (p *PharmacyProduct) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkProductDeletable(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_location_create.go`:

```go
// Create es la regla de negocio de crear pharmacy_location: el nombre no puede repetirse.
func (p *PharmacyLocation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkLocationNameFree(tx, p.Name); err != nil {
		return 0, err
	}
	p.UpdatedAt = time.NowNano()
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_location_update.go`:

```go
// Update es la regla de negocio de actualizar pharmacy_location: si cambia el nombre, nadie más puede tenerlo.
func (next *PharmacyLocation) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyLocation, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	if err := applyLocationFields(&merged, next, fields); err != nil {
		return 0, err
	}
	if merged.Name != old.Name {
		if err := checkLocationNameFree(tx, merged.Name); err != nil {
			return 0, err
		}
	}
	merged.UpdatedAt = time.NowNano()
	if err := tx.UpdateFields(&merged, old.Id, []string{PharmacyLocation_.UpdatedAt}); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_location_delete.go`:

```go
// Delete es la regla de negocio de borrar pharmacy_location: una ubicación con stock no se borra.
func (p *PharmacyLocation) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkLocationDeletable(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```

`rule_pharmacy_operation_create.go` — creating the row IS doing the operation; `actor_id`, `movement_ids` and `created_at` are server fields:

```go
// Create es la regla de negocio de crear pharmacy_operation: ejecuta la operación de stock (receive, entry, output o
// transfer) dentro de la transacción y deja en la fila quién la hizo y los ids de los movimientos que produjo. Si el
// hecho de costo o la alerta de una salida no se pueden publicar, la operación completa se deshace.
func (p *PharmacyOperation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	eff, err := m.runOperation(tx, p, a.UserID)
	if err != nil {
		return 0, err
	}
	p.ActorId = a.UserID
	p.MovementIds = movementIds(eff.movements)
	p.CreatedAt = time.NowNano()
	if err := m.announce(eff); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
```

`rule_expiring_stock_read.go` and `rule_low_stock_read.go` — read models; the filter is the receiver-like `filter` (D52). `expiringOn` and `lowStockOn` are written in Stage 5 (the methods `ListExpiring` and `ListLowStock` call the same functions):

```go
// Read es la regla de negocio de leer expiring_stock: el stock que caduca dentro de filter.WithinDays días.
func (p *ExpiringStock) Read(a crud.CurrentUser, filter *ExpiringStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.expiringOn(m.store, filter.WithinDays)
	if err != nil {
		return nil, err
	}
	list := ExpiringStockList{}
	for i := range rows {
		list = append(list, &rows[i])
	}
	return &list, nil
}

// Read es la regla de negocio de leer low_stock: los productos activos en o por debajo de su mínimo.
func (p *LowStock) Read(a crud.CurrentUser, filter *LowStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.lowStockOn(m.store)
	if err != nil {
		return nil, err
	}
	list := LowStockList{}
	for i := range rows {
		list = append(list, &rows[i])
	}
	return &list, nil
}
```
If `<Entity>List` is not a `[]*Entity` slice in the generated file, build the list the way `service_catalog/rule_currency_option_read.go` does (https://github.com/veltylabs/service_catalog/blob/main/rule_currency_option_read.go). Each rule file imports what it uses (`webtyp.com/crud`, `webtyp.com/model`, `webtyp.com/time`).

The package does not compile yet (Stage 5). Commit and push; **open the pull request now** with what exists and keep committing to it.
