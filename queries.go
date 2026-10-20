package pharmacymanager

import (
	"slices"
)

// defaultMovementLimit is how many movements list_movements returns when the caller gives no limit.
const defaultMovementLimit = 100

// ListBatches returns the batches of a product.
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

// batchIDsOf returns the ids of the batches of a product.
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

// ListStock returns the stock levels with quantity > 0, optionally of one product and/or one location.
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

// ListMovements returns the ledger, newest first, optionally of one product or one batch and between two Unix seconds
// (a zero bound is open). Without a limit it returns defaultMovementLimit entries.
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

// GetBatch returns one batch of the tenant.
func (m *Module) GetBatch(tenantId, id string) (PharmacyBatch, error) {
	return getBatch(m.db, tenantId, id)
}
