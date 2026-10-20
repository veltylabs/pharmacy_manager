package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/events/mock"
	"webtyp.com/money"
	"webtyp.com/orm"
	"webtyp.com/stock"
	"webtyp.com/storage/mem"
	"webtyp.com/time"
)

// output builds the arguments of a RegisterOutput.
func output(productId, locationId string, t stock.Movement, quantity float64) pharmacymanager.RegisterOutputArgs {
	return pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: productId, LocationId: locationId, Quantity: quantity, MovementType: string(t),
	}
}

func TestRegisterOutput_DrawsTheEarliestExpiryFirstAndSplits(t *testing.T) {
	m := newModule(t)
	p, loc := newProduct(t, m, "p1"), newLocation(t, m, "Store")
	soon, later := time.NowUnix()+10*time.Day, time.NowUnix()+60*time.Day
	for _, r := range []pharmacymanager.ReceiveStockArgs{receive(p.Id, loc.Id, "LATER", later, 10), receive(p.Id, loc.Id, "SOON", soon, 4)} {
		if _, err := m.ReceiveStock(r); err != nil {
			t.Fatal(err)
		}
	}

	mvs, err := m.RegisterOutput(output(p.Id, loc.Id, stock.Sale, 6))
	if err != nil {
		t.Fatalf("output: %v", err)
	}
	if len(mvs) != 2 || mvs[0].Quantity != 4 || mvs[1].Quantity != 2 {
		t.Fatalf("SOON (4) must be drained first, then 2 from LATER; got %+v", mvs)
	}
	levels, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id})
	if len(levels) != 1 || levels[0].Quantity != 8 {
		t.Errorf("8 must remain in LATER, got %+v", levels)
	}
}

func TestRegisterOutput_Rules(t *testing.T) {
	m := newModule(t)
	p, loc := newProduct(t, m, "p1"), newLocation(t, m, "Store")
	notForSale, err := m.CreateProduct(pharmacymanager.PharmacyProduct{
		TenantId: testTenant, Sku: "nfs", Name: "Not for sale", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxExempt, IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	expired := time.NowUnix() - 10*time.Day
	future := time.NowUnix() + 30*time.Day
	for _, r := range []pharmacymanager.ReceiveStockArgs{
		receive(p.Id, loc.Id, "OLD", expired, 5), receive(p.Id, loc.Id, "NEW", future, 5), receive(notForSale.Id, loc.Id, "X", future, 5),
	} {
		if _, err := m.ReceiveStock(r); err != nil {
			t.Fatal(err)
		}
	}
	oldBatch := batchIDOf(t, m, p.Id, "OLD")

	if _, err := m.RegisterOutput(output(notForSale.Id, loc.Id, stock.Sale, 1)); err != pharmacymanager.ErrNotForSale {
		t.Errorf("selling a product that is not for sale: got %v, want ErrNotForSale", err)
	}
	if _, err := m.RegisterOutput(output(p.Id, loc.Id, stock.Sale, 6)); err != pharmacymanager.ErrInsufficientStock {
		t.Errorf("only 5 usable (the expired batch does not count): got %v, want ErrInsufficientStock", err)
	}
	if levels, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id}); len(levels) != 2 {
		t.Errorf("a failed output must change nothing, got %+v", levels)
	}
	forced := output(p.Id, loc.Id, stock.Sale, 1)
	forced.BatchId = oldBatch
	if _, err := m.RegisterOutput(forced); err != pharmacymanager.ErrBatchExpired {
		t.Errorf("forcing an expired batch: got %v, want ErrBatchExpired", err)
	}
	forced.MovementType = string(stock.AdjustOut)
	if _, err := m.RegisterOutput(forced); err != nil {
		t.Errorf("a write-off may take expired stock: %v", err)
	}
	if _, err := m.RegisterOutput(output(p.Id, loc.Id, stock.Purchase, 1)); err != pharmacymanager.ErrMovementType {
		t.Errorf("an inbound type in an output: got %v, want ErrMovementType", err)
	}
}

func batchIDOf(t *testing.T, m *pharmacymanager.Module, productId, number string) string {
	t.Helper()
	batches, err := m.ListBatches(testTenant, productId)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range batches {
		if b.BatchNumber == number {
			return b.Id
		}
	}
	t.Fatalf("batch %s not found", number)
	return ""
}

// recordingSink is a money.Sink that keeps what it accepts.
type recordingSink struct{ facts []money.Fact }

func (s *recordingSink) Accept(f money.Fact) error { s.facts = append(s.facts, f); return nil }

func TestRegisterOutput_PublishesTheCostAtCost(t *testing.T) {
	broker := &mock.Broker{}
	sink := &recordingSink{}
	if err := money.Subscribe(broker, sink, func(err error) { t.Errorf("money: %v", err) }); err != nil {
		t.Fatal(err)
	}
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker, MoneyScope: "pharmacy"})
	if err != nil {
		t.Fatal(err)
	}
	p, loc := newProduct(t, m, "p1"), newLocation(t, m, "Store")
	args := receive(p.Id, loc.Id, "L1", time.NowUnix()+30*time.Day, 10)
	args.UnitCost = 333
	if _, err := m.ReceiveStock(args); err != nil {
		t.Fatal(err)
	}
	if len(sink.facts) != 0 {
		t.Fatalf("a purchase is not a fact of this module, got %+v", sink.facts)
	}

	if _, err := m.RegisterOutput(output(p.Id, loc.Id, stock.Sale, 2.5)); err != nil {
		t.Fatal(err)
	}
	if len(sink.facts) != 1 {
		t.Fatalf("one Consumed fact expected, got %+v", sink.facts)
	}
	f := sink.facts[0]
	if f.Kind != money.Consumed || f.Side != money.Outflow || f.Scope != "pharmacy" || f.Amount.Amount != 833 || f.Amount.Currency != money.CLP {
		t.Errorf("333 × 2.5 = 832.5 must round to 833 CLP, got %+v", f)
	}
}
