package pharmacymanager

import (
	"slices"
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// defaultMovementLimit es cuántos movimientos devuelve list_movements cuando el llamador no da un límite.
const defaultMovementLimit = 100

// ListBatches devuelve los lotes de un producto.
// Ejemplo: batches, err := module.ListBatches("tenant1", "prod1")

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

func compareText(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func page[T any](list []T, offset, limit int) []T {
	if offset > 0 {
		if offset >= len(list) {
			return nil
		}
		list = list[offset:]
	}
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}
