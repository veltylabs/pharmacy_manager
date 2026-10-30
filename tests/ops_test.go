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




type saveLocationsReq struct{ Records []pharmacymanager.PharmacyLocation }
func (a *saveLocationsReq) ModelName() string { return "save_locations_req" }
func (a *saveLocationsReq) Schema() []model.Field { return nil }
func (a *saveLocationsReq) Pointers() []any { return nil }
func (a *saveLocationsReq) IsNil() bool { return a == nil }
func (a *saveLocationsReq) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}

type saveProductsReq struct{ Records []pharmacymanager.PharmacyProduct }
func (a *saveProductsReq) ModelName() string { return "save_products_req" }
func (a *saveProductsReq) Schema() []model.Field { return nil }
func (a *saveProductsReq) Pointers() []any { return nil }
func (a *saveProductsReq) IsNil() bool { return a == nil }
func (a *saveProductsReq) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}

type deleteIdsReq struct{ Ids []string }
func (a *deleteIdsReq) ModelName() string { return "delete_ids_req" }
func (a *deleteIdsReq) Schema() []model.Field { return nil }
func (a *deleteIdsReq) Pointers() []any { return nil }
func (a *deleteIdsReq) IsNil() bool { return a == nil }
func (a *deleteIdsReq) EncodeFields(w model.FieldWriter) {
	arr := w.Array("ids", len(a.Ids))
	for _, id := range a.Ids {
		arr.String(id)
	}
}


// call ejecuta una operación a través del caller en proceso, de la forma en que lo haría un transporte, y devuelve su error.
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
	if err := call(t, c, pharmacymanager.OpUpsertLocation, &saveLocationsReq{Records: []pharmacymanager.PharmacyLocation{
		{TenantId: testTenant, Name: "Store", IsActive: true},
	}}, &loc); err != nil || loc.Id == "" {
		t.Fatalf("upsert_location: %v, %+v", err, loc)
	}

	var prod pharmacymanager.PharmacyProduct
	if err := call(t, c, pharmacymanager.OpUpsertProduct, &saveProductsReq{Records: []pharmacymanager.PharmacyProduct{
		{TenantId: testTenant, Sku: "p1", Name: "Paracetamol 500 mg", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 1500, IsActive: true},
	}}, &prod); err != nil || prod.Id == "" {
		t.Fatalf("upsert_product: %v, %+v", err, prod)
	}

	var mv pharmacymanager.PharmacyMovement
	recv := receive(prod.Id, loc.Id, "L1", time.NowUnix()+30*time.Day, 10)
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

	// la actualización de la vista: ids + los campos a cambiar + un registro con los valores
	if err := call(t, c, pharmacymanager.OpUpdateProduct, &pharmacymanager.UpdateProductArgs{
		Ids: []string{prod.Id}, Fields: []string{"name"}, Record: pharmacymanager.PharmacyProduct{Name: "Paracetamol 1 g"},
	}, nil); err != nil {
		t.Fatalf("update_product: %v", err)
	}
	if got, err := m.GetProduct(testTenant, prod.Id); err != nil || got.Name != "Paracetamol 1 g" || got.Price != 1500 {
		t.Errorf("only the name must change: %+v, %v", got, err)
	}

	// la eliminación de la vista: solo ids; un producto con lotes no puede ir
	err := call(t, c, pharmacymanager.OpDeleteProduct, &deleteIdsReq{Ids: []string{prod.Id}}, nil)
	if err == nil || !strings.Contains(err.Error(), string(pharmacymanager.ErrProductInUse)) {
		t.Errorf("deleting a product with batches: got %v, want ErrProductInUse", err)
	}
	if err := call(t, c, pharmacymanager.OpDeleteLocation, &deleteIdsReq{Ids: []string{loc.Id}}, nil); err == nil || !strings.Contains(err.Error(), string(pharmacymanager.ErrLocationInUse)) {
		t.Errorf("deleting a location with stock: got %v, want ErrLocationInUse", err)
	}
}
