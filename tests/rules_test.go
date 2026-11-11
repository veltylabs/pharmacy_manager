package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/time"
	"webtyp.com/stock"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

// call runs one operation of the module as `user` through the same loopback the demo uses and answers the callback error.
func call(user string, pub events.Publisher, mod *pharmacymanager.Module, entity string, action model.Action, args model.Encodable, into model.Decodable) error {
	var callErr error
	crud.LoopbackActingAs(pub, user, mod).Call(
		router.Operation{Module: pharmacymanager.ModelName, Entity: entity, Action: action},
		args, into, func(err error) { callErr = err })
	return callErr
}

// failedField answers the field a refusal points at, or "".
func failedField(err error) string {
	if f, ok := err.(interface{ FailedField() string }); ok {
		return f.FailedField()
	}
	return ""
}

// operate creates one pharmacy_operation as u1 and answers the callback error.
func operate(m *pharmacymanager.Module, op *pharmacymanager.PharmacyOperation) error {
	return call("u1", nil, m, "pharmacy_operation", model.Create, &crud.CreateArgs{Records: []model.Model{op}}, nil)
}

// inDays answers the Unix second `days` days from now.
func inDays(days int) time.Unix { return time.NowUnix() + time.Unix(days)*time.Day }

func TestProduct_CreateThroughCrud(t *testing.T) {
	m := newModule(t)
	err := call("u1", nil, m, "pharmacy_product", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyProduct{Sku: "p1", Name: "P", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true}}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prods, err := m.ListProducts(pharmacymanager.ListProductsArgs{TenantId: testTenant})
	if err != nil || len(prods) != 1 || prods[0].TenantId != testTenant {
		t.Fatalf("expected 1 product with TenantId %v, got %v (err: %v)", testTenant, prods, err)
	}
}

func TestProduct_CreateRules(t *testing.T) {
	m := newModule(t)
	err := call("u1", nil, m, "pharmacy_product", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyProduct{Sku: "p1", Name: "P", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true, ForSale: true}}}, nil)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("expected ErrorInvalid for for_sale without price, got %v (kind %v)", err, model.KindOf(err))
	}

	newProduct(t, m, "p2")
	err = call("u1", nil, m, "pharmacy_product", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyProduct{Sku: "p2", Name: "P2", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true}}}, nil)
	if model.KindOf(err) != model.ErrorConflict {
		t.Errorf("expected ErrorConflict for existing sku, got %v", err)
	}
}

func TestProduct_UpdateZeroesPriceWhenNotForSale(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	err := call("u1", nil, m, "pharmacy_product", model.Update, &crud.UpdateArgs{IDs: []string{p.Id}, Fields: []string{"for_sale"}, Record: &pharmacymanager.PharmacyProduct{ForSale: false}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	updated, _ := m.GetProduct(testTenant, p.Id)
	if updated.Price != 0 || updated.ForSale != false {
		t.Errorf("expected price 0 and for_sale false, got price %v and for_sale %v", updated.Price, updated.ForSale)
	}
}

func TestProduct_UpdateSkuTaken(t *testing.T) {
	m := newModule(t)
	p1 := newProduct(t, m, "p1")
	p2 := newProduct(t, m, "p2")
	err := call("u1", nil, m, "pharmacy_product", model.Update, &crud.UpdateArgs{IDs: []string{p2.Id}, Fields: []string{"sku"}, Record: &pharmacymanager.PharmacyProduct{Sku: p1.Sku}}, nil)
	if model.KindOf(err) != model.ErrorConflict {
		t.Errorf("expected ErrorConflict, got %v", err)
	}
}

func TestProduct_DeleteWithBatchesRefused(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")
	m.ReceiveStock(pharmacymanager.ReceiveStockArgs{TenantId: testTenant, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})

	err := call("u1", nil, m, "pharmacy_product", model.Delete, &crud.DeleteArgs{IDs: []string{p.Id}}, nil)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("expected ErrorInvalid, got %v", err)
	}

	p2 := newProduct(t, m, "p2")
	err2 := call("u1", nil, m, "pharmacy_product", model.Delete, &crud.DeleteArgs{IDs: []string{p2.Id}}, nil)
	if err2 != nil {
		t.Errorf("expected no error deleting product with no stock, got %v", err2)
	}
}

func TestLocation_DeleteWithStockRefused(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")
	m.ReceiveStock(pharmacymanager.ReceiveStockArgs{TenantId: testTenant, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})

	err := call("u1", nil, m, "pharmacy_location", model.Delete, &crud.DeleteArgs{IDs: []string{loc.Id}}, nil)
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("expected ErrorInvalid, got %v", err)
	}

	loc2 := newLocation(t, m, "Empty")
	err2 := call("u1", nil, m, "pharmacy_location", model.Delete, &crud.DeleteArgs{IDs: []string{loc2.Id}}, nil)
	if err2 != nil {
		t.Errorf("expected no error deleting empty location, got %v", err2)
	}
}

func TestLocation_UpdateNameTaken(t *testing.T) {
	m := newModule(t)
	locA := newLocation(t, m, "A")
	locB := newLocation(t, m, "B")

	err := call("u1", nil, m, "pharmacy_location", model.Update, &crud.UpdateArgs{IDs: []string{locB.Id}, Fields: []string{"name"}, Record: &pharmacymanager.PharmacyLocation{Name: locA.Name}}, nil)
	if model.KindOf(err) != model.ErrorConflict {
		t.Errorf("expected ErrorConflict, got %v", err)
	}
}

func TestOperation_ReceiveCreatesBatchAndLevel(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")

	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, _ := m.ListBatches(testTenant, p.Id)
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %v", len(batches))
	}

	levels, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id})
	if len(levels) != 1 || levels[0].Quantity != 10 {
		t.Errorf("expected 1 stock level of quantity 10, got %v", levels)
	}

	movs, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: p.Id})
	if len(movs) != 1 || movs[0].MovementType != string(stock.Purchase) {
		t.Errorf("expected 1 movement of type PURCHASE, got %v", movs)
	}

	var list pharmacymanager.PharmacyOperationList
	err = call("u1", nil, m, "pharmacy_operation", model.Read, &crud.ReadArgs{}, &list)
	if err != nil || len(list) != 1 || list[0].ActorId != "u1" || list[0].MovementIds != movs[0].Id {
		t.Errorf("expected operation to have ActorId 'u1' and MovementIds matched, got err: %v, items: %v", err, list)
	}
}

func TestOperation_OutputFirstExpiryFirst(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")
	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L5", ExpirationDate: inDays(5), Quantity: 5})
	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L2", ExpirationDate: inDays(2), Quantity: 5})

	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationOutput, ProductId: p.Id, LocationId: loc.Id, Quantity: 3, MovementType: string(stock.Sale)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, _ := m.ListBatches(testTenant, p.Id)
	var l2Batch string
	for _, b := range batches {
		if b.BatchNumber == "L2" {
			l2Batch = b.Id
		}
	}

	movs, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: p.Id, BatchId: l2Batch})

	var saleMov *pharmacymanager.PharmacyMovement
	for i, m := range movs {
		if m.MovementType == string(stock.Sale) {
			saleMov = &movs[i]
		}
	}

	if saleMov == nil || saleMov.Quantity != 3 {
		t.Errorf("expected 1 movement of type SALE on batch L2 of quantity 3, got %v", saleMov)
	}
}

func TestOperation_OutputNeverNegative(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")
	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(5), Quantity: 5})

	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationOutput, ProductId: p.Id, LocationId: loc.Id, Quantity: 6, MovementType: string(stock.Sale)})
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("expected ErrorInvalid, got %v", err)
	}

	levels, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id})
	if len(levels) != 1 || levels[0].Quantity != 5 {
		t.Errorf("expected 1 stock level of quantity 5, got %v", levels)
	}

	movs, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: p.Id})
	if len(movs) != 1 {
		t.Errorf("expected 1 movement, got %v", movs)
	}
}

func TestOperation_TransferMovesBetweenLocations(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	locA := newLocation(t, m, "A")
	locB := newLocation(t, m, "B")

	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: locA.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})

	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationTransfer, ProductId: p.Id, LocationId: locA.Id, ToLocationId: locB.Id, Quantity: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	levels, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id})
	var qtyA, qtyB float64
	for _, l := range levels {
		if l.LocationId == locA.Id {
			qtyA += l.Quantity
		}
		if l.LocationId == locB.Id {
			qtyB += l.Quantity
		}
	}

	if qtyA != 6 || qtyB != 4 {
		t.Errorf("expected level A to be 6 and level B to be 4, got A: %v, B: %v", qtyA, qtyB)
	}

	movs, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: p.Id})
	var outMov, inMov bool
	for _, m := range movs {
		if m.MovementType == string(stock.TransferOut) {
			outMov = true
		}
		if m.MovementType == string(stock.TransferIn) {
			inMov = true
		}
	}

	if !outMov || !inMov {
		t.Errorf("expected TRANSFER_OUT and TRANSFER_IN movements, got %v", movs)
	}
}

func TestOperation_UnknownKindRejected(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")
	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: "x", ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 10})
	if failedField(err) != "kind" {
		t.Errorf("expected failedField to be 'kind', got %v", failedField(err))
	}
}

func TestOperation_ServerFieldsRefused(t *testing.T) {
	m := newModule(t)
	err := operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ActorId: "x"})
	if model.KindOf(err) != model.ErrorInvalid {
		t.Errorf("expected ErrorInvalid, got %v", err)
	}
	batches, _ := m.ListBatches(testTenant, "")
	if len(batches) != 0 {
		t.Errorf("expected no batches, got %v", len(batches))
	}
}

func TestMovement_NoWriteOperations(t *testing.T) {
	m := newModule(t)
	err := call("u1", nil, m, "pharmacy_movement", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyMovement{}}}, nil)
	if err == nil || failedField(err) != "" && err.Error() == "unknown operation" {
		t.Errorf("expected error containing 'unknown operation', got %v", err)
	}
}

func TestReadModel_LowStock(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	p.MinQuantity = 10
	p.UpdatedAt = 0
	p.MinQuantity = 10
	m.UpdateProduct(p)
	loc := newLocation(t, m, "Store")

	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(30), Quantity: 5})

	var list pharmacymanager.LowStockList
	err := call("u1", nil, m, "low_stock", model.Read, &crud.ReadArgs{Filter: &pharmacymanager.LowStock{}}, &list)
	if err != nil || len(list) != 1 || list[0].Total != 5 {
		t.Fatalf("expected 1 low stock row with Total 5, got err: %v, items: %v", err, list)
	}

	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L2", ExpirationDate: inDays(30), Quantity: 20})
	list = nil
	call("u1", nil, m, "low_stock", model.Read, &crud.ReadArgs{Filter: &pharmacymanager.LowStock{}}, &list)
	if len(list) != 0 {
		t.Errorf("expected 0 low stock rows after receiving more stock, got %v", len(list))
	}
}

func TestReadModel_ExpiringWithinDays(t *testing.T) {
	m := newModule(t)
	p := newProduct(t, m, "p1")
	loc := newLocation(t, m, "Store")

	operate(m, &pharmacymanager.PharmacyOperation{Kind: pharmacymanager.OperationReceive, ProductId: p.Id, LocationId: loc.Id, BatchNumber: "L1", ExpirationDate: inDays(5), Quantity: 5})

	var list pharmacymanager.ExpiringStockList
	err := call("u1", nil, m, "expiring_stock", model.Read, &crud.ReadArgs{Filter: &pharmacymanager.ExpiringStock{WithinDays: 30}}, &list)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 expiring stock row, got err: %v, items: %v", err, list)
	}

	list = nil
	call("u1", nil, m, "expiring_stock", model.Read, &crud.ReadArgs{Filter: &pharmacymanager.ExpiringStock{WithinDays: 1}}, &list)
	if len(list) != 0 {
		t.Errorf("expected 0 expiring stock rows within 1 day, got %v", len(list))
	}
}

func TestCreate_PublishesChange(t *testing.T) {
	broker := &recordingPublisher{}
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{TenantID: testTenant, IDs: &testIDGen{}, Publisher: broker})
	if err != nil {
		t.Fatal(err)
	}

	err = call("u1", broker, m, "pharmacy_product", model.Create, &crud.CreateArgs{Records: []model.Model{&pharmacymanager.PharmacyProduct{Sku: "p1", Name: "P", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, IsActive: true}}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(broker.events) != 1 {
		t.Fatalf("expected 1 event published, got %v", len(broker.events))
	}
	if _, ok := broker.events[0].Payload.(*crud.Change); !ok {
		t.Errorf("expected Payload to be *crud.Change, got %T", broker.events[0].Payload)
	}
}
