package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
)

func TestTenantIsolation(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	rec, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tenant2 := "tenant-2"

	products, _ := m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: tenant2})
	if len(products) != 0 {
		t.Errorf("expected empty products list, got %v", len(products))
	}

	stocks, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: tenant2, ProductId: product.Id})
	if len(stocks) != 0 {
		t.Errorf("expected empty stock list, got %v", len(stocks))
	}

	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: tenant2, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L2", Quantity: 5,
	})
	if err != pharmacymanager.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}

	_, err = m.GetBatch(tenant2, rec.BatchId)
	if err != pharmacymanager.ErrBatchNotFound {
		t.Errorf("expected ErrBatchNotFound, got %v", err)
	}
}
