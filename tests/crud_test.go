package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
)

func TestCreateProduct_StoresIt(t *testing.T) {
	m := newModule(t)
	created := newProduct(t, m, "a1")
	got, err := m.GetProduct(testTenant, created.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sku != created.Sku || got.Name != created.Name || got.Unit != created.Unit {
		t.Errorf("expected %v %v %v, got %v %v %v", created.Sku, created.Name, created.Unit, got.Sku, got.Name, got.Unit)
	}
	if got.UpdatedAt <= 0 {
		t.Errorf("expected UpdatedAt > 0, got %v", got.UpdatedAt)
	}
}

func TestCreateProduct_SkuIsUniquePerTenant(t *testing.T) {
	m := newModule(t)
	_ = newProduct(t, m, "a1")

	// Create again in same tenant
	_, err := m.CreateProduct(pharmacymanager.PharmacyProduct{TaxTreatment: pharmacymanager.TaxTaxed, TenantId: testTenant, Sku: "a1", Name: "Same", Unit: "BOX", IsActive: true})
	if err != pharmacymanager.ErrSkuExists {
		t.Errorf("expected ErrSkuExists, got %v", err)
	}

	// Create in different tenant
	created2, err2 := m.CreateProduct(pharmacymanager.PharmacyProduct{TaxTreatment: pharmacymanager.TaxTaxed, TenantId: "tenant-2", Sku: "a1", Name: "Same", Unit: "BOX", IsActive: true})
	if err2 != nil {
		t.Errorf("expected nil error, got %v", err2)
	}
	if created2.Id == "" {
		t.Errorf("expected product to be created in tenant-2")
	}
}

func TestGetProduct_ForeignTenant(t *testing.T) {
	m := newModule(t)
	created := newProduct(t, m, "a1")

	_, err := m.GetProduct("tenant-2", created.Id)
	if err != pharmacymanager.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestUpdateProduct(t *testing.T) {
	m := newModule(t)
	a1 := newProduct(t, m, "a1")
	a2 := newProduct(t, m, "a2")

	// Update name of a1
	a1.Name = "New Name"
	updated, err := m.UpdateProduct(a1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name to change")
	}

	// Update a1 to have a2's sku
	a1.Sku = a2.Sku
	_, err = m.UpdateProduct(a1)
	if err != pharmacymanager.ErrSkuExists {
		t.Errorf("expected ErrSkuExists, got %v", err)
	}

	// Update unknown id
	unknown := a1
	unknown.Id = "unknown"
	unknown.Sku = "a3"
	_, err = m.UpdateProduct(unknown)
	if err != pharmacymanager.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestPatchProducts(t *testing.T) {
	m := newModule(t)
	a1 := newProduct(t, m, "a1")

	err := m.PatchProducts([]string{a1.Id}, []string{"name"}, pharmacymanager.PharmacyProduct{Name: "Patched Name", Unit: "BAG"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	got, _ := m.GetProduct(testTenant, a1.Id)
	if got.Name != "Patched Name" || got.Unit != "BOX" {
		t.Errorf("expected name Patched Name and unit BOX, got %v and %v", got.Name, got.Unit)
	}

	err = m.PatchProducts([]string{a1.Id}, []string{"bogus"}, pharmacymanager.PharmacyProduct{Name: "Patched Name"})
	if vErr, ok := err.(pharmacymanager.ValidationError); !ok || vErr.Err != pharmacymanager.ErrUnknownField {
		t.Errorf("expected ValidationError with ErrUnknownField, got %v", err)
	}
}

func TestDeactivateProduct(t *testing.T) {
	m := newModule(t)
	a1 := newProduct(t, m, "a1")
	l1 := newLocation(t, m, "Loc1")

	err := m.DeactivateProduct(testTenant, a1.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	products, err := m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: testTenant, ActiveOnly: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, product := range products {
		if product.Id == a1.Id {
			t.Errorf("expected deactivated product to not be listed")
		}
	}

	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: a1.Id, LocationId: l1.Id, BatchNumber: "B1", Quantity: 10,
	})
	if err != pharmacymanager.ErrProductInactive {
		t.Errorf("expected ErrProductInactive, got %v", err)
	}
}

func TestDeleteProducts(t *testing.T) {
	m := newModule(t)
	a1 := newProduct(t, m, "empty")
	a2 := newProduct(t, m, "with-batch")
	l1 := newLocation(t, m, "Loc1")

	_, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: a2.Id, LocationId: l1.Id, BatchNumber: "B1", Quantity: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = m.DeleteProducts([]string{a1.Id})
	if err != nil {
		t.Errorf("unexpected error deleting empty product: %v", err)
	}
	_, err = m.GetProduct(testTenant, a1.Id)
	if err != pharmacymanager.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}

	err = m.DeleteProducts([]string{a2.Id})
	if err != pharmacymanager.ErrProductInUse {
		t.Errorf("expected ErrProductInUse, got %v", err)
	}
	_, err = m.GetProduct(testTenant, a2.Id)
	if err != nil {
		t.Errorf("expected product to still be there, got error: %v", err)
	}

	err = m.DeleteProducts([]string{"unknown"})
	if err != pharmacymanager.ErrProductNotFound {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestListProducts_PagedAndFiltered(t *testing.T) {
	m := newModule(t)
	// A, B, C created in order; B deactivated
	a := pharmacymanager.PharmacyProduct{TaxTreatment: pharmacymanager.TaxTaxed, TenantId: testTenant, Sku: "1", Name: "A", Unit: "BOX", IsActive: true}
	a, _ = m.CreateProduct(a)
	b := pharmacymanager.PharmacyProduct{TaxTreatment: pharmacymanager.TaxTaxed, TenantId: testTenant, Sku: "2", Name: "B", Unit: "BOX", IsActive: true}
	b, _ = m.CreateProduct(b)
	c := pharmacymanager.PharmacyProduct{TaxTreatment: pharmacymanager.TaxTaxed, TenantId: testTenant, Sku: "3", Name: "C", Unit: "BOX", IsActive: true}
	c, _ = m.CreateProduct(c)

	_ = m.DeactivateProduct(testTenant, b.Id)

	products, err := m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: testTenant, Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(products) != 2 || products[0].Name != "B" || products[1].Name != "C" {
		t.Errorf("expected B, C, got %v", products)
	}

	productsActive, err := m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: testTenant, ActiveOnly: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(productsActive) != 2 || productsActive[0].Name != "A" || productsActive[1].Name != "C" {
		t.Errorf("expected A, C, got %v", productsActive)
	}
}

func TestProductExists(t *testing.T) {
	m := newModule(t)
	active := newProduct(t, m, "active")
	deact := newProduct(t, m, "deact")
	_ = m.DeactivateProduct(testTenant, deact.Id)

	exists, err := m.ProductExists(testTenant, active.Id)
	if !exists || err != nil {
		t.Errorf("expected true, nil, got %v, %v", exists, err)
	}

	exists, err = m.ProductExists(testTenant, deact.Id)
	if exists || err != nil {
		t.Errorf("expected false, nil, got %v, %v", exists, err)
	}

	exists, err = m.ProductExists(testTenant, "unknown")
	if exists || err != nil {
		t.Errorf("expected false, nil, got %v, %v", exists, err)
	}

	exists, err = m.ProductExists("tenant-2", active.Id)
	if exists || err != nil {
		t.Errorf("expected false, nil, got %v, %v", exists, err)
	}
}

func TestCreateProduct_Rules(t *testing.T) {
	m := newModule(t)
	base := func() pharmacymanager.PharmacyProduct {
		return pharmacymanager.PharmacyProduct{TenantId: testTenant, Sku: "r1", Name: "N", Unit: pharmacymanager.UnitBox,
			TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true}
	}

	p := base()
	p.ForSale, p.Price = true, 0
	if _, err := m.CreateProduct(p); err != pharmacymanager.ErrSaleNeedsPrice {
		t.Errorf("for sale without price: got %v, want ErrSaleNeedsPrice", err)
	}
	p = base()
	p.Unit = "BAG"
	if _, err := m.CreateProduct(p); err != pharmacymanager.ErrUnit {
		t.Errorf("unknown unit: got %v, want ErrUnit", err)
	}
	p = base()
	p.TaxTreatment = ""
	if _, err := m.CreateProduct(p); err != pharmacymanager.ErrTaxTreatmentRequired {
		t.Errorf("no tax treatment: got %v, want ErrTaxTreatmentRequired", err)
	}
	p = base()
	p.Name = ""
	if _, err := m.CreateProduct(p); err == nil {
		t.Error("an empty name must be refused")
	} else if _, ok := err.(pharmacymanager.ValidationError); !ok {
		t.Errorf("empty name: got %T %v, want a ValidationError", err, err)
	}
	p = base()
	p.ForSale, p.Price = false, 500
	created, err := m.CreateProduct(p)
	if err != nil {
		t.Fatalf("not for sale with a price: %v", err)
	}
	if created.Price != 0 {
		t.Errorf("a product not for sale must be stored with price 0, got %d", created.Price)
	}
}

func TestProductCard(t *testing.T) {
	m := newModule(t)
	product, err := m.CreateProduct(pharmacymanager.PharmacyProduct{TenantId: testTenant, Sku: "PARA500", Name: "Paracetamol 500 mg",
		Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 1500, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}

	name, unit, tax, price, active, err := m.ProductCard(testTenant, product.Id)
	if err != nil || name != "Paracetamol 500 mg" || unit != pharmacymanager.UnitBox || tax != pharmacymanager.TaxTaxed || price != 1500 || !active {
		t.Errorf("got %q %q %q %d %v %v", name, unit, tax, price, active, err)
	}
	if _, _, _, _, _, err := m.ProductCard(testTenant, "unknown"); err != pharmacymanager.ErrNotFound {
		t.Errorf("unknown product: got %v, want ErrNotFound", err)
	}
}

func TestCreateLocation_NameIsUniquePerTenant(t *testing.T) {
	m := newModule(t)
	_ = newLocation(t, m, "Store")

	_, err := m.CreateLocation(pharmacymanager.PharmacyLocation{TenantId: testTenant, Name: "Store", IsActive: true})
	if err != pharmacymanager.ErrLocationNameExists {
		t.Errorf("expected ErrLocationNameExists, got %v", err)
	}

	loc2, err2 := m.CreateLocation(pharmacymanager.PharmacyLocation{TenantId: "tenant-2", Name: "Store", IsActive: true})
	if err2 != nil {
		t.Errorf("expected nil error, got %v", err2)
	}
	if loc2.Id == "" {
		t.Errorf("expected location to be created in tenant-2")
	}
}

func TestUpdateLocation(t *testing.T) {
	m := newModule(t)
	store := newLocation(t, m, "Store")
	_ = newLocation(t, m, "Counter")

	store.Name = "Counter"
	_, err := m.UpdateLocation(store)
	if err != pharmacymanager.ErrLocationNameExists {
		t.Errorf("expected ErrLocationNameExists, got %v", err)
	}

	store.Name = "Warehouse"
	updated, err := m.UpdateLocation(store)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updated.Name != "Warehouse" {
		t.Errorf("expected name to change to Warehouse")
	}
}

func TestDeleteLocations(t *testing.T) {
	m := newModule(t)
	empty := newLocation(t, m, "Empty")
	holding := newLocation(t, m, "Holding")
	product := newProduct(t, m, "i1")

	_, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: holding.Id, BatchNumber: "B1", Quantity: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = m.DeleteLocations([]string{empty.Id})
	if err != nil {
		t.Errorf("unexpected error deleting empty location: %v", err)
	}

	err = m.DeleteLocations([]string{holding.Id})
	if err != pharmacymanager.ErrLocationInUse {
		t.Errorf("expected ErrLocationInUse, got %v", err)
	}
}

func TestInactiveLocationRefusesReceipts(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	loc.IsActive = false
	_, err := m.UpdateLocation(loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "B1", Quantity: 10,
	})
	if err != pharmacymanager.ErrLocationInactive {
		t.Errorf("expected ErrLocationInactive, got %v", err)
	}
}
func init() {
	// dummy use of newLocation just to avoid unused
}
