package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"github.com/veltylabs/pharmacy_manager/seed"
)

func TestSeed_LoadsValidatedDemoData(t *testing.T) {
	m := newModule(t)
	data, err := seed.Load(m, testTenant)
	if err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	if len(data.Locations) != 2 || len(data.Products) != 3 {
		t.Fatalf("2 locations and 3 products expected, got %d and %d", len(data.Locations), len(data.Products))
	}
	levels, err := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant})
	if err != nil || len(levels) != 4 {
		t.Fatalf("3 products received in the store plus 10 units moved to the counter make 4 levels, got %d, %v", len(levels), err)
	}
}
