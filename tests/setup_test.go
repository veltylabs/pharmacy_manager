package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

const testTenant = "tenant-1"

// testIDGen is the test double of model.IDGenerator: a module never builds its own generator.
type testIDGen struct{ n int }

func (g *testIDGen) NewID() string {
	g.n++
	return "id-" + fmt.Convert(g.n).String()
}

// newModule is a Module over an in-memory database, without events.
func newModule(t *testing.T) *pharmacymanager.Module {
	t.Helper()
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// newProduct creates an active, taxed, for-sale product (price 1000) and fails the test if it cannot.
func newProduct(t *testing.T, m *pharmacymanager.Module, sku string) pharmacymanager.PharmacyProduct {
	t.Helper()
	p, err := m.CreateProduct(pharmacymanager.PharmacyProduct{
		TenantId: testTenant, Sku: sku, Name: "Product " + sku, Unit: pharmacymanager.UnitBox,
		TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 1000, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProduct(%s): %v", sku, err)
	}
	return p
}

// newLocation creates an active location and fails the test if it cannot.
func newLocation(t *testing.T, m *pharmacymanager.Module, name string) pharmacymanager.PharmacyLocation {
	t.Helper()
	l, err := m.CreateLocation(pharmacymanager.PharmacyLocation{TenantId: testTenant, Name: name, IsActive: true})
	if err != nil {
		t.Fatalf("CreateLocation(%s): %v", name, err)
	}
	return l
}
