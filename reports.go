package pharmacymanager

import (
	"slices"
	"webtyp.com/crud"

	"webtyp.com/fmt"
	"webtyp.com/notify"
	"webtyp.com/stock"
	"webtyp.com/time"
)

// ReviewExpiring lanza una alerta cuando los lotes con existencias han caducado o caducan dentro de withinDays (días),
// y los devuelve. La aplicación lo llama una vez al día; este módulo no tiene programador.
// Ejemplo: expiring, err := module.ReviewExpiring("tenant1", 30)
func (m *Module) ReviewExpiring(tenantId string, withinDays int) ([]ExpiringStock, error) {
	rows, err := m.ListExpiring(ListExpiringArgs{TenantId: tenantId, WithinDays: int64(withinDays)})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return rows, nil
	}
	var expired, soon int
	for _, row := range rows {
		if row.Expired {
			expired++
		} else {
			soon++
		}
	}
	alertErr := m.raise(notify.Alert{
		Code:    AlertExpiring,
		Subject: textExpiringName,
		Body:    fmt.Sprintf(msgExpiring, expired, soon, withinDays),
		Source:  notify.Ref{Type: alertSourceReview, ID: time.FormatISO8601(time.NowUnix())[:10]},
	})
	if alertErr != nil {
		return rows, alertErr
	}
	return rows, nil
}

// ListExpiring devuelve las existencias (cantidad > 0) cuyo lote ha caducado o caduca dentro de WithinDays (días), los más próximos primero.
// La regla es stock.Lot.ExpiresWithin; este módulo no la repite.
// Ejemplo: expiring, err := module.ListExpiring(ListExpiringArgs{TenantId: "t1", WithinDays: 30})

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
