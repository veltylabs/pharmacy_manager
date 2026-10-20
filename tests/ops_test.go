package tests

import (
	"strings"
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/router/loopback"
	"webtyp.com/stock"
	"webtyp.com/time"
)

// call runs one operation through the in-process caller, the way a transport would, and returns its error.
func call(t *testing.T, c router.Caller, op string, args model.Encodable, out model.Decodable) error {
	t.Helper()
	var got error
	c.Call(pharmacymanager.ModelName+"."+op, args, out, func(err error) { got = err })
	return got
}

func TestOperations_ThroughTheCaller(t *testing.T) {
	m := newModule(t)
	c := loopback.New(m)

	var loc pharmacymanager.PharmacyLocation
	if err := call(t, c, pharmacymanager.OpUpsertLocation, &pharmacymanager.UpsertLocationArgs{Records: []pharmacymanager.PharmacyLocation{
		{TenantId: testTenant, Name: "Store", IsActive: true},
	}}, &loc); err != nil || loc.Id == "" {
		t.Fatalf("upsert_location: %v, %+v", err, loc)
	}

	var prod pharmacymanager.PharmacyProduct
	if err := call(t, c, pharmacymanager.OpUpsertProduct, &pharmacymanager.UpsertProductArgs{Records: []pharmacymanager.PharmacyProduct{
		{TenantId: testTenant, Sku: "p1", Name: "Paracetamol 500 mg", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 1500, IsActive: true},
	}}, &prod); err != nil || prod.Id == "" {
		t.Fatalf("upsert_product: %v, %+v", err, prod)
	}

	var mv pharmacymanager.PharmacyMovement
	recv := receive(prod.Id, loc.Id, "L1", time.Now()+30*stock.SecondsPerDay, 10)
	if err := call(t, c, pharmacymanager.OpReceiveStock, &recv, &mv); err != nil || mv.Quantity != 10 {
		t.Fatalf("receive_stock: %v, %+v", err, mv)
	}

	var levels pharmacymanager.PharmacyStockLevelList
	if err := call(t, c, pharmacymanager.OpListStock, &pharmacymanager.ListStockArgs{TenantId: testTenant}, &levels); err != nil || len(levels) != 1 || levels[0].Quantity != 10 {
		t.Fatalf("list_stock: %v, %d rows", err, len(levels))
	}

	out := output(prod.Id, loc.Id, stock.Sale, 4)
	var sold pharmacymanager.PharmacyMovementList
	if err := call(t, c, pharmacymanager.OpRegisterOutput, &out, &sold); err != nil || len(sold) != 1 || sold[0].Quantity != 4 {
		t.Fatalf("register_output: %v, %d movements", err, len(sold))
	}

	// the view's update: ids + the fields to change + a record with the values
	if err := call(t, c, pharmacymanager.OpUpdateProduct, &pharmacymanager.UpdateProductArgs{
		Ids: []string{prod.Id}, Fields: []string{"name"}, Record: pharmacymanager.PharmacyProduct{Name: "Paracetamol 1 g"},
	}, nil); err != nil {
		t.Fatalf("update_product: %v", err)
	}
	if got, err := m.GetProduct(testTenant, prod.Id); err != nil || got.Name != "Paracetamol 1 g" || got.Price != 1500 {
		t.Errorf("only the name must change: %+v, %v", got, err)
	}

	// the view's delete: ids only; a product with batches cannot go
	err := call(t, c, pharmacymanager.OpDeleteProduct, &pharmacymanager.DeleteIdsArgs{Ids: []string{prod.Id}}, nil)
	if err == nil || !strings.Contains(err.Error(), string(pharmacymanager.ErrProductInUse)) {
		t.Errorf("deleting a product with batches: got %v, want ErrProductInUse", err)
	}
	if err := call(t, c, pharmacymanager.OpDeleteLocation, &pharmacymanager.DeleteIdsArgs{Ids: []string{loc.Id}}, nil); err == nil || !strings.Contains(err.Error(), string(pharmacymanager.ErrLocationInUse)) {
		t.Errorf("deleting a location with stock: got %v, want ErrLocationInUse", err)
	}
}
