package pharmacymanager

import (
	"slices"
)

// defaultMovementLimit es cuántos movimientos devuelve list_movements cuando el llamador no da un límite.
const defaultMovementLimit = 100

// ListBatches devuelve los lotes de un producto.
// Ejemplo: batches, err := module.ListBatches("tenant1", "prod1")
func (m *Module) ListBatches(tenantId, productId string) ([]PharmacyBatch, error) {
	var proxy PharmacyBatch
	qb := m.db.Query(&proxy).Where(PharmacyBatch_.TenantId).Eq(tenantId).Where(PharmacyBatch_.ProductId).Eq(productId)
	list, err := ReadAllPharmacyBatch(qb)
	if err != nil {
		return nil, err
	}
	return derefBatches(list), nil
}

func derefBatches(list PharmacyBatchList) []PharmacyBatch {
	out := make([]PharmacyBatch, len(list))
	for i, b := range list {
		out[i] = *b
	}
	return out
}

// batchIDsOf devuelve los ids de los lotes de un producto.
func (m *Module) batchIDsOf(tenantId, productId string) ([]string, error) {
	batches, err := m.ListBatches(tenantId, productId)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(batches))
	for i, b := range batches {
		ids[i] = b.Id
	}
	return ids, nil
}

// ListStock devuelve los niveles de existencias con cantidad > 0, opcionalmente de un producto y/o una ubicación.
// Ejemplo: stock, err := module.ListStock(ListStockArgs{TenantId: "t1", ProductId: "p1"})
func (m *Module) ListStock(a ListStockArgs) ([]PharmacyStockLevel, error) {
	var proxy PharmacyStockLevel
	qb := m.db.Query(&proxy).Where(PharmacyStockLevel_.TenantId).Eq(a.TenantId)
	if a.LocationId != "" {
		qb = qb.Where(PharmacyStockLevel_.LocationId).Eq(a.LocationId)
	}
	list, err := ReadAllPharmacyStockLevel(qb)
	if err != nil {
		return nil, err
	}
	var batchIDs []string
	if a.ProductId != "" {
		if batchIDs, err = m.batchIDsOf(a.TenantId, a.ProductId); err != nil {
			return nil, err
		}
	}
	var out []PharmacyStockLevel
	for _, l := range list {
		if l.Quantity <= 0 {
			continue
		}
		if a.ProductId != "" && !slices.Contains(batchIDs, l.BatchId) {
			continue
		}
		out = append(out, *l)
	}
	return out, nil
}

// ListMovements devuelve el libro mayor, los más nuevos primero, opcionalmente de un producto o un lote y entre dos segundos Unix
// (un límite cero está abierto). Sin un límite, devuelve defaultMovementLimit entradas.
// Ejemplo: movs, err := module.ListMovements(ListMovementsArgs{TenantId: "t1", Limit: 10})
func (m *Module) ListMovements(a ListMovementsArgs) ([]PharmacyMovement, error) {
	var proxy PharmacyMovement
	qb := m.db.Query(&proxy).Where(PharmacyMovement_.TenantId).Eq(a.TenantId)
	if a.BatchId != "" {
		qb = qb.Where(PharmacyMovement_.BatchId).Eq(a.BatchId)
	}
	list, err := ReadAllPharmacyMovement(qb)
	if err != nil {
		return nil, err
	}
	var batchIDs []string
	if a.ProductId != "" {
		if batchIDs, err = m.batchIDsOf(a.TenantId, a.ProductId); err != nil {
			return nil, err
		}
	}
	var out []PharmacyMovement
	for _, mv := range list {
		if a.ProductId != "" && !slices.Contains(batchIDs, mv.BatchId) {
			continue
		}
		if (a.From > 0 && mv.CreatedAt < a.From) || (a.To > 0 && mv.CreatedAt > a.To) {
			continue
		}
		out = append(out, *mv)
	}
	slices.Reverse(out) // el almacén devuelve el orden de inserción: invertido, es el desempate de las entradas escritas en el mismo segundo
	slices.SortStableFunc(out, func(x, y PharmacyMovement) int {
		switch {
		case x.CreatedAt > y.CreatedAt:
			return -1
		case x.CreatedAt < y.CreatedAt:
			return 1
		}
		return 0
	})
	limit := int(a.Limit)
	if limit <= 0 {
		limit = defaultMovementLimit
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// GetBatch devuelve un lote del inquilino.
// Ejemplo: batch, err := module.GetBatch("tenant1", "batch1")
func (m *Module) GetBatch(tenantId, id string) (PharmacyBatch, error) {
	return getBatch(m.db, tenantId, id)
}

// ListProducts devuelve los productos de un inquilino, por nombre. active_only oculta los desactivados; limit 0 = sin límite.
// Ejemplo: prods, err := module.ListProducts(ListProductsArgs{TenantId: "t1", Limit: 50})
func (m *Module) ListProducts(a ListProductsArgs) ([]PharmacyProduct, error) {
	var proxy PharmacyProduct
	qb := m.db.Query(&proxy).Where(PharmacyProduct_.TenantId).Eq(a.TenantId)
	if a.ActiveOnly {
		qb = qb.Where(PharmacyProduct_.IsActive).Eq(true)
	}
	list, err := ReadAllPharmacyProduct(qb)
	if err != nil {
		return nil, err
	}
	out := make([]PharmacyProduct, len(list))
	for i, p := range list {
		out[i] = *p
	}
	slices.SortFunc(out, func(x, y PharmacyProduct) int { return compareText(x.Name, y.Name) })
	return page(out, int(a.Offset), int(a.Limit)), nil
}

// ListLocations devuelve las ubicaciones de un inquilino, por nombre.
// Ejemplo: locs, err := module.ListLocations(ListLocationsArgs{TenantId: "t1"})
func (m *Module) ListLocations(a ListLocationsArgs) ([]PharmacyLocation, error) {
	var proxy PharmacyLocation
	qb := m.db.Query(&proxy).Where(PharmacyLocation_.TenantId).Eq(a.TenantId)
	if a.ActiveOnly {
		qb = qb.Where(PharmacyLocation_.IsActive).Eq(true)
	}
	list, err := ReadAllPharmacyLocation(qb)
	if err != nil {
		return nil, err
	}
	out := make([]PharmacyLocation, len(list))
	for i, l := range list {
		out[i] = *l
	}
	slices.SortFunc(out, func(x, y PharmacyLocation) int { return compareText(x.Name, y.Name) })
	return out, nil
}

// compareText ordena dos cadenas por orden de bytes.
func compareText(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// page devuelve el segmento desde offset (desplazamiento), de longitud máxima limit (límite); limit 0 = hasta el final.
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
