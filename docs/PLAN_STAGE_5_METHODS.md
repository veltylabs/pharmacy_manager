← [Stage 4](PLAN_STAGE_4_RULES.md) | Next → [Stage 6](PLAN_STAGE_6_BROWSER.md)

# Stage 5 — every kept method on the store; `ops.go` deleted

Other modules and apps call these methods: **every exported method keeps its name, parameters and result types, in the file where it is today.** Only the body changes (table of Stage 2.4). A method that takes a `tenantId` gets its store from `m.storeFor(tenantId)` (another tenant sees nothing: «not found» / empty; an empty tenant answers `ErrTenantRequired`); the methods that take only ids (`PatchProducts`, `DeleteProducts`, `PatchLocations`, `DeleteLocations`) use `m.store`. Each is written below; copy the doc comment (with its `Ejemplo:`) of today's method above it.

## 5.1 `module.go`

First delete the OLD copies that Stage 3 replaced: `getProduct`, `getLocation`, `getBatch`, `batchByNumber`, `levelAt`, `levelOrCreate`, `applyToLevel`, `addMovement`, `lotsAt`, `productBySKU`, and the OLD bodies of `ReceiveStock`, `RegisterEntry`, `RegisterOutput`, `TransferStock`. Keep `Deps`, `Module`, `New`, `checkProduct`, `publish`, `publishCosts` and the constants. Imports of `module.go` after the stage: `webtyp.com/crud`, `events`, `fmt`, `model`, `money`, `orm`, `time` (drop `notify` and `stock` if unused: `announce` lives in `stock_tx.go`). Then write:

```go
// productColumns son las columnas que cambian al actualizar un producto completo (todas menos id y tenant_id).
var productColumns = []string{PharmacyProduct_.Sku, PharmacyProduct_.Name, PharmacyProduct_.Unit, PharmacyProduct_.TaxTreatment,
	PharmacyProduct_.ForSale, PharmacyProduct_.Price, PharmacyProduct_.Controlled, PharmacyProduct_.Barcode,
	PharmacyProduct_.RegistryNumber, PharmacyProduct_.MinQuantity, PharmacyProduct_.IsActive, PharmacyProduct_.UpdatedAt}

func (m *Module) CreateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	if p.Id == "" {
		p.Id = m.ids.NewID()
	}
	p.UpdatedAt = time.NowNano()
	if err := checkProduct(&p); err != nil {
		return PharmacyProduct{}, err
	}
	if err := p.Validate(model.ActionCreate); err != nil {
		return PharmacyProduct{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		if err := checkSkuFree(tx, p.Sku); err != nil {
			return err
		}
		return tx.Create(&p)
	})
	if err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

func (m *Module) GetProduct(tenantId, id string) (PharmacyProduct, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	return getProduct(store, id)
}

func (m *Module) CreateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	if l.Id == "" {
		l.Id = m.ids.NewID()
	}
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionCreate); err != nil {
		return PharmacyLocation{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		if err := checkLocationNameFree(tx, l.Name); err != nil {
			return err
		}
		return tx.Create(&l)
	})
	if err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

func (m *Module) GetLocation(tenantId, id string) (PharmacyLocation, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	return getLocation(store, id)
}

// runAtomic ejecuta una operación de stock en una transacción del tenant dado y responde sus efectos sin publicarlos.
func (m *Module) runAtomic(tenantId string, op *PharmacyOperation, actor string) (*opEffects, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return nil, err
	}
	var eff *opEffects
	err = store.Atomic(func(tx crud.Tx) error {
		var err error
		eff, err = m.runOperation(tx, op, actor)
		return err
	})
	return eff, err
}

func (m *Module) ReceiveStock(a ReceiveStockArgs) (PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationReceive, ProductId: a.ProductId, LocationId: a.LocationId, BatchNumber: a.BatchNumber,
		ExpirationDate: a.ExpirationDate, UnitCost: a.UnitCost, Quantity: a.Quantity,
		SourceDocumentType: a.SourceDocumentType, SourceDocumentId: a.SourceDocumentId, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return PharmacyMovement{}, err
	}
	if err := m.announce(eff); err != nil {
		return eff.movements[0], err
	}
	return eff.movements[0], nil
}

func (m *Module) RegisterEntry(a RegisterEntryArgs) (PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationEntry, ProductId: a.ProductId, LocationId: a.LocationId, BatchId: a.BatchId,
		Quantity: a.Quantity, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return PharmacyMovement{}, err
	}
	if err := m.announce(eff); err != nil {
		return eff.movements[0], err
	}
	return eff.movements[0], nil
}

// RegisterOutput: si el hecho de costo o la alerta no se pueden publicar después de guardar, los movimientos se devuelven
// CON el error: el cambio de stock ya está guardado (como siempre).
func (m *Module) RegisterOutput(a RegisterOutputArgs) ([]PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationOutput, ProductId: a.ProductId, LocationId: a.LocationId, BatchId: a.BatchId,
		Quantity: a.Quantity, MovementType: a.MovementType, SourceDocumentType: a.SourceDocumentType,
		SourceDocumentId: a.SourceDocumentId, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	return eff.movements, m.announce(eff)
}

func (m *Module) TransferStock(a TransferStockArgs) ([]PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationTransfer, ProductId: a.ProductId, LocationId: a.FromLocationId,
		ToLocationId: a.ToLocationId, Quantity: a.Quantity, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	return eff.movements, m.announce(eff)
}
```
Delete `func (m *Module) ModelName() string` if still there (generated).

## 5.2 `crud.go` — replace the whole file body (same exported methods)

Delete `productByID` and `locationByID` (they ignored the tenant: the sealed `m.store` replaces them). Write:

```go
// locationColumns son las columnas que cambian al actualizar una ubicación completa.
var locationColumns = []string{PharmacyLocation_.Name, PharmacyLocation_.IsSalesPoint, PharmacyLocation_.IsActive, PharmacyLocation_.UpdatedAt}

// updateProductTx guarda el producto p sobre el guardado existing: es lo que comparten UpdateProduct y PatchProducts.
func updateProductTx(tx crud.Tx, existing, p *PharmacyProduct) error {
	p.UpdatedAt = time.NowNano()
	if err := checkProduct(p); err != nil {
		return err
	}
	if err := p.Validate(model.ActionUpdate); err != nil {
		return err
	}
	if p.Sku != existing.Sku {
		if err := checkSkuFree(tx, p.Sku); err != nil {
			return err
		}
	}
	return tx.UpdateFields(p, p.Id, productColumns)
}

// updateLocationTx es updateProductTx de las ubicaciones.
func updateLocationTx(tx crud.Tx, existing, l *PharmacyLocation) error {
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionUpdate); err != nil {
		return err
	}
	if l.Name != existing.Name {
		if err := checkLocationNameFree(tx, l.Name); err != nil {
			return err
		}
	}
	return tx.UpdateFields(l, l.Id, locationColumns)
}

func (m *Module) UpdateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		existing, err := getProduct(tx, p.Id)
		if err != nil {
			return err
		}
		return updateProductTx(tx, &existing, &p)
	})
	if err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

func (m *Module) UpsertProduct(p PharmacyProduct) (PharmacyProduct, error) {
	if p.Id == "" {
		return m.CreateProduct(p)
	}
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	if _, err := getProduct(store, p.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateProduct(p)
	} else if err != nil {
		return PharmacyProduct{}, err
	}
	return m.UpdateProduct(p)
}

func (m *Module) PatchProducts(ids, fields []string, rec PharmacyProduct) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			current, err := getProduct(tx, id)
			if err != nil {
				return err
			}
			existing := current
			if err := applyProductFields(&current, &rec, fields); err != nil {
				return err
			}
			if err := updateProductTx(tx, &existing, &current); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) DeactivateProduct(tenantId, id string) error {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return err
	}
	return store.Atomic(func(tx crud.Tx) error {
		p, err := getProduct(tx, id)
		if err != nil {
			return err
		}
		p.IsActive = false
		p.UpdatedAt = time.NowNano()
		return tx.UpdateFields(&p, p.Id, []string{PharmacyProduct_.IsActive, PharmacyProduct_.UpdatedAt})
	})
}

func (m *Module) DeleteProducts(ids []string) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			p, err := getProduct(tx, id)
			if err != nil {
				return err
			}
			if err := checkProductDeletable(tx, p.Id); err != nil {
				return err
			}
			if err := tx.Delete(&p, p.Id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) UpdateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		existing, err := getLocation(tx, l.Id)
		if err != nil {
			return err
		}
		return updateLocationTx(tx, &existing, &l)
	})
	if err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

func (m *Module) UpsertLocation(l PharmacyLocation) (PharmacyLocation, error) {
	if l.Id == "" {
		return m.CreateLocation(l)
	}
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	if _, err := getLocation(store, l.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateLocation(l)
	} else if err != nil {
		return PharmacyLocation{}, err
	}
	return m.UpdateLocation(l)
}

func (m *Module) PatchLocations(ids, fields []string, rec PharmacyLocation) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			current, err := getLocation(tx, id)
			if err != nil {
				return err
			}
			existing := current
			if err := applyLocationFields(&current, &rec, fields); err != nil {
				return err
			}
			if err := updateLocationTx(tx, &existing, &current); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) DeleteLocations(ids []string) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			l, err := getLocation(tx, id)
			if err != nil {
				return err
			}
			if err := checkLocationDeletable(tx, l.Id); err != nil {
				return err
			}
			if err := tx.Delete(&l, l.Id); err != nil {
				return err
			}
		}
		return nil
	})
}
```
Imports of `crud.go`: `webtyp.com/crud`, `webtyp.com/model`, `webtyp.com/time`. The old `PatchProducts` answered a field error `<unknown>` for an unknown field; now it names the field (`applyProductFields`): a test that asserts `"<unknown>"` is corrected to the field's name.

## 5.3 `queries.go` — the readers (keep `defaultMovementLimit`, `compareText`, `page` untouched; delete `derefBatches`, `listProducts`, `listLocations`)

```go
// batchesOn responde los lotes de un producto.
func batchesOn(tx crud.Tx, productId string) ([]PharmacyBatch, error) {
	rows, err := allRows(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, productId))
	if err != nil {
		return nil, err
	}
	out := make([]PharmacyBatch, len(rows))
	for i, r := range rows {
		out[i] = *r.(*PharmacyBatch)
	}
	return out, nil
}

// batchIDsOn responde los ids de los lotes de un producto.
func batchIDsOn(tx crud.Tx, productId string) ([]string, error) {
	batches, err := batchesOn(tx, productId)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(batches))
	for i, b := range batches {
		ids[i] = b.Id
	}
	return ids, nil
}

func (m *Module) ListBatches(tenantId, productId string) ([]PharmacyBatch, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return nil, err
	}
	return batchesOn(store, productId)
}

func (m *Module) GetBatch(tenantId, id string) (PharmacyBatch, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return PharmacyBatch{}, err
	}
	return getBatch(store, id)
}

// stockOn responde los niveles con cantidad > 0, opcionalmente de un producto y/o una ubicación.
func stockOn(tx crud.Tx, productId, locationId string) ([]PharmacyStockLevel, error) {
	var conds []storage.Condition
	if locationId != "" {
		conds = append(conds, storage.Eq(PharmacyStockLevel_.LocationId, locationId))
	}
	rows, err := allRows(tx, func() model.Model { return &PharmacyStockLevel{} }, conds...)
	if err != nil {
		return nil, err
	}
	var batchIDs []string
	if productId != "" {
		if batchIDs, err = batchIDsOn(tx, productId); err != nil {
			return nil, err
		}
	}
	var out []PharmacyStockLevel
	for _, r := range rows {
		l := r.(*PharmacyStockLevel)
		if l.Quantity <= 0 {
			continue
		}
		if productId != "" && !containsString(batchIDs, l.BatchId) {
			continue
		}
		out = append(out, *l)
	}
	return out, nil
}

func (m *Module) ListStock(a ListStockArgs) ([]PharmacyStockLevel, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	return stockOn(store, a.ProductId, a.LocationId)
}

// ListMovements: los más nuevos primero. El filtro va en la consulta (el tope de una lectura es crud.MaxPage filas).
func (m *Module) ListMovements(a ListMovementsArgs) ([]PharmacyMovement, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	var conds []storage.Condition
	if a.BatchId != "" {
		conds = append(conds, storage.Eq(PharmacyMovement_.BatchId, a.BatchId))
	}
	if a.ProductId != "" {
		ids, err := batchIDsOn(store, a.ProductId)
		if err != nil || len(ids) == 0 {
			return nil, err
		}
		conds = append(conds, storage.In(PharmacyMovement_.BatchId, ids))
	}
	if a.From > 0 {
		conds = append(conds, storage.Gte(PharmacyMovement_.CreatedAt, int64(a.From)))
	}
	if a.To > 0 {
		conds = append(conds, storage.Lte(PharmacyMovement_.CreatedAt, int64(a.To)))
	}
	limit := a.Limit
	if limit <= 0 {
		limit = defaultMovementLimit
	}
	if limit > crud.MaxPage {
		limit = crud.MaxPage
	}
	var out []PharmacyMovement
	err = store.ReadAll(func() model.Model { return &PharmacyMovement{} },
		crud.Query{Conditions: conds, Order: []storage.Order{storage.Desc(PharmacyMovement_.CreatedAt), storage.Desc(PharmacyMovement_.Id)}, Limit: int64(limit)},
		func(r model.Model) { out = append(out, *r.(*PharmacyMovement)) })
	return out, err
}

// productsOn responde los productos, por nombre; activeOnly oculta los desactivados.
func productsOn(tx crud.Tx, activeOnly bool) ([]PharmacyProduct, error) {
	var conds []storage.Condition
	if activeOnly {
		conds = append(conds, storage.Eq(PharmacyProduct_.IsActive, true))
	}
	rows, err := allRows(tx, func() model.Model { return &PharmacyProduct{} }, conds...)
	if err != nil {
		return nil, err
	}
	out := make([]PharmacyProduct, len(rows))
	for i, r := range rows {
		out[i] = *r.(*PharmacyProduct)
	}
	slices.SortFunc(out, func(x, y PharmacyProduct) int { return compareText(x.Name, y.Name) })
	return out, nil
}

func (m *Module) ListProducts(a ListProductsArgs) ([]PharmacyProduct, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	out, err := productsOn(store, a.ActiveOnly)
	if err != nil {
		return nil, err
	}
	return page(out, int(a.Offset), int(a.Limit)), nil
}

func (m *Module) ListLocations(a ListLocationsArgs) ([]PharmacyLocation, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	var conds []storage.Condition
	if a.ActiveOnly {
		conds = append(conds, storage.Eq(PharmacyLocation_.IsActive, true))
	}
	rows, err := allRows(store, func() model.Model { return &PharmacyLocation{} }, conds...)
	if err != nil {
		return nil, err
	}
	out := make([]PharmacyLocation, len(rows))
	for i, r := range rows {
		out[i] = *r.(*PharmacyLocation)
	}
	slices.SortFunc(out, func(x, y PharmacyLocation) int { return compareText(x.Name, y.Name) })
	return out, nil
}
```
Imports of `queries.go`: `slices`, `webtyp.com/crud`, `webtyp.com/model`, `webtyp.com/storage`. If a ledger test depended on the order of two movements written in the same nanosecond, make that test compare by `MovementType`/`BatchId` instead of position and say so in the PR (the store now orders by id).

## 5.4 `reports.go` and `alerts.go`

`ReviewExpiring` and the doc comments stay; replace the bodies below (imports of `reports.go`: `slices`, `webtyp.com/crud`, `fmt`, `notify`, `stock`, `time`):

```go
func (m *Module) ListExpiring(a ListExpiringArgs) ([]ExpiringStock, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	return m.expiringOn(store, a.WithinDays)
}

// expiringOn lo comparten ListExpiring y la regla de leer expiring_stock: el stock cuyo lote ya caducó o caduca dentro de
// withinDays días, los más próximos primero. La regla es stock.Lot.ExpiresWithin.
func (m *Module) expiringOn(tx crud.Tx, withinDays int64) ([]ExpiringStock, error) {
	if withinDays < 0 {
		return nil, ErrDays
	}
	now := time.NowUnix()
	levels, err := stockOn(tx, "", "")
	if err != nil {
		return nil, err
	}
	var out []ExpiringStock
	for _, level := range levels {
		batch, err := getBatch(tx, level.BatchId)
		if err != nil {
			return nil, err
		}
		lot := stock.Lot{ID: batch.Id, Expiry: batch.ExpirationDate, Quantity: level.Quantity}
		if !lot.ExpiresWithin(now, int(withinDays)) {
			continue
		}
		product, err := getProduct(tx, batch.ProductId)
		if err != nil {
			return nil, err
		}
		out = append(out, ExpiringStock{
			ProductId: product.Id, ProductName: product.Name, BatchId: batch.Id, BatchNumber: batch.BatchNumber,
			LocationId: level.LocationId, Quantity: level.Quantity, ExpirationDate: batch.ExpirationDate, Expired: lot.Expired(now),
		})
	}
	slices.SortStableFunc(out, func(x, y ExpiringStock) int {
		switch {
		case x.ExpirationDate < y.ExpirationDate:
			return -1
		case x.ExpirationDate > y.ExpirationDate:
			return 1
		}
		return 0
	})
	return out, nil
}

func (m *Module) ListLowStock(a ListLowStockArgs) ([]LowStock, error) {
	store, err := m.storeFor(a.TenantId)
	if err != nil {
		return nil, err
	}
	return m.lowStockOn(store)
}

// lowStockOn lo comparten ListLowStock y la regla de leer low_stock: los productos activos con mínimo (> 0) cuyo total
// está en o por debajo de él, por nombre.
func (m *Module) lowStockOn(tx crud.Tx) ([]LowStock, error) {
	products, err := productsOn(tx, true)
	if err != nil {
		return nil, err
	}
	var out []LowStock
	for _, p := range products {
		levels, err := stockOn(tx, p.Id, "")
		if err != nil {
			return nil, err
		}
		var total float64
		for _, level := range levels {
			total += level.Quantity
		}
		if stock.Low(total, p.MinQuantity) {
			out = append(out, LowStock{ProductId: p.Id, ProductName: p.Name, Total: total, MinQuantity: p.MinQuantity})
		}
	}
	return out, nil
}

func (m *Module) ProductExists(tenantId, productId string) (bool, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return false, err
	}
	p, err := getProduct(store, productId)
	if err == ErrProductNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return p.IsActive, nil
}

func (m *Module) ProductCard(tenantId, productId string) (name, unit, taxTreatment string, price int64, active bool, err error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return "", "", "", 0, false, err
	}
	p, err := getProduct(store, productId)
	if err == ErrProductNotFound {
		return "", "", "", 0, false, ErrNotFound
	}
	if err != nil {
		return "", "", "", 0, false, err
	}
	return p.Name, p.Unit, p.TaxTreatment, p.Price, p.IsActive, nil
}
```
`alerts.go`: delete `totalProduct` (`totalOf` of Stage 3 replaces it; `grep -rn totalProduct .` → empty). Everything else of `alerts.go` stays.

## 5.5 Deletions (each with its criterion)

| Delete | Criterion |
|---|---|
| `ops.go` (the `Op*` constants, `MountOperations`, every `op*` handler) | `test ! -e ops.go`; `grep -rn "OpListProducts\|MountOperations(reg" --include=*.go . \| grep -v definition_gen` → empty |
| `func productByID`, `locationByID`, `derefBatches`, `listProducts`, `listLocations`, `totalProduct` | `grep -rn "productByID\|locationByID\|derefBatches\|listProducts(\|listLocations(\|totalProduct" --include=*.go .` → empty |
| in `errors.go`: `ErrNoRecords`, `ErrNoIds`, `ErrCannotRead` — ONLY if the grep shows no use outside the files this stage deletes | `grep -rn "ErrNoRecords\|ErrNoIds\|ErrCannotRead" --include=*.go . \| grep -v "errors.go\|_test.go"` → empty |

`definitionc` wrote the new `MountOperations()`; never write one by hand. `gotest` may still fail on `view.go`, `ui/` and old tests (Stages 6–7). Commit and push.
