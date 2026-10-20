package pharmacymanager

import (
	"slices"

	"webtyp.com/stock"
	"webtyp.com/time"
)

// ListExpiring returns the stock (quantity > 0) whose batch is expired or expires within WithinDays days, soonest first.
// The rule is stock.Lot.ExpiresWithin; this module does not repeat it.
func (m *Module) ListExpiring(a ListExpiringArgs) ([]ExpiringStock, error) {
	if a.WithinDays < 0 {
		return nil, ErrDays
	}
	now := time.Now()
	levels, err := m.ListStock(ListStockArgs{TenantId: a.TenantId})
	if err != nil {
		return nil, err
	}
	var out []ExpiringStock
	for _, level := range levels {
		batch, err := getBatch(m.db, a.TenantId, level.BatchId)
		if err != nil {
			return nil, err
		}
		lot := stock.Lot{ID: batch.Id, Expiry: batch.ExpirationDate, Quantity: level.Quantity}
		if !lot.ExpiresWithin(now, int(a.WithinDays)) {
			continue
		}
		product, err := getProduct(m.db, a.TenantId, batch.ProductId)
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

// ListLowStock returns the active products with a minimum quantity (> 0) whose total, over every location and batch, is at
// or below it, by name. A later alert module calls it; this module raises no alert.
func (m *Module) ListLowStock(a ListLowStockArgs) ([]LowStock, error) {
	products, err := m.ListProducts(ListProductsArgs{TenantId: a.TenantId, ActiveOnly: true})
	if err != nil {
		return nil, err
	}
	var out []LowStock
	for _, p := range products {
		if p.MinQuantity <= 0 {
			continue
		}
		levels, err := m.ListStock(ListStockArgs{TenantId: a.TenantId, ProductId: p.Id})
		if err != nil {
			return nil, err
		}
		var total float64
		for _, level := range levels {
			total += level.Quantity
		}
		if total <= p.MinQuantity {
			out = append(out, LowStock{ProductId: p.Id, ProductName: p.Name, Total: total, MinQuantity: p.MinQuantity})
		}
	}
	return out, nil
}

// ProductExists tells whether a product exists in the tenant and is active. Other modules declare the same method in a
// narrow interface of their own; *Module satisfies it without importing them.
func (m *Module) ProductExists(tenantId, productId string) (bool, error) {
	p, err := getProduct(m.db, tenantId, productId)
	if err == ErrProductNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return p.IsActive, nil
}

// ProductCard returns what another module snapshots about a product. An absent or foreign product is ErrNotFound.
func (m *Module) ProductCard(tenantId, productId string) (name, unit, taxTreatment string, price int64, active bool, err error) {
	p, err := getProduct(m.db, tenantId, productId)
	if err == ErrProductNotFound {
		return "", "", "", 0, false, ErrNotFound
	}
	if err != nil {
		return "", "", "", 0, false, err
	}
	return p.Name, p.Unit, p.TaxTreatment, p.Price, p.IsActive, nil
}
