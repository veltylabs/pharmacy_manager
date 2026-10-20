package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/events"
	"webtyp.com/events/mock"
	"webtyp.com/money"
	"webtyp.com/orm"
	"webtyp.com/stock"
	"webtyp.com/storage/mem"
)

func TestEvents_OnePerMovement(t *testing.T) {
	broker := &mock.Broker{}
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker})
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	broker.Subscribe(pharmacymanager.TopicMovementRegistered, func(events.Event) { n++ })

	product := newProduct(t, m, "i1")
	locA := newLocation(t, m, "A")
	locB := newLocation(t, m, "B")

	_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, BatchNumber: "B1", Quantity: 5,
	})
	_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, BatchNumber: "B2", Quantity: 5,
	})

	if n != 2 {
		t.Errorf("expected 2 events, got %v", n)
	}

	_, _ = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, Quantity: 7, MovementType: string(stock.InternalUse),
	})

	if n != 4 {
		t.Errorf("expected 4 events, got %v", n)
	}

	_, _ = m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: product.Id, FromLocationId: locA.Id, ToLocationId: locB.Id, Quantity: 1,
	})

	if n != 6 {
		t.Errorf("expected 6 events, got %v", n)
	}
}

func TestEvents_NilPublisherIsSilent(t *testing.T) {
	m := newModule(t) // no publisher
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	_, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "B1", Quantity: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, Quantity: 2, MovementType: string(stock.InternalUse),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvents_FailedOperationPublishesNothing(t *testing.T) {
	broker := &mock.Broker{}
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker})
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	broker.Subscribe(pharmacymanager.TopicMovementRegistered, func(events.Event) { n++ })

	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "B1", Quantity: 5,
	})
	nStart := n

	_, err = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, Quantity: 6, MovementType: string(stock.InternalUse),
	})
	if err != pharmacymanager.ErrInsufficientStock {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}

	if n != nStart {
		t.Errorf("expected counter did not move, got %v vs %v", n, nStart)
	}
}

func TestCostFacts_OnlyWithScopeAndCost(t *testing.T) {
	broker := &mock.Broker{}
	sink := &recordingSink{}
	if err := money.Subscribe(broker, sink, func(err error) { t.Errorf("money: %v", err) }); err != nil {
		t.Fatal(err)
	}

	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker})
	if err != nil {
		t.Fatal(err)
	}

	product := newProduct(t, m, "i1")
	locA := newLocation(t, m, "A")

	// output with MoneyScope: ""
	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, BatchNumber: "B1", Quantity: 5, UnitCost: 100,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, Quantity: 1, MovementType: string(stock.InternalUse),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// output of a batch with cost 0 and MoneyScope: "x"
	m2, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker, MoneyScope: "x"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	product2 := newProduct(t, m2, "i2")
	locC := newLocation(t, m2, "C")
	_, err = m2.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product2.Id, LocationId: locC.Id, BatchNumber: "B2", Quantity: 5, UnitCost: 0,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m2.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product2.Id, LocationId: locC.Id, Quantity: 1, MovementType: string(stock.InternalUse),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// transfer and entry with cost and scope
	m3, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}, Publisher: broker, MoneyScope: "x"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	product3 := newProduct(t, m3, "i3")
	locD := newLocation(t, m3, "D")
	locE := newLocation(t, m3, "E")
	rec3, err := m3.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product3.Id, LocationId: locD.Id, BatchNumber: "B3", Quantity: 5, UnitCost: 100,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m3.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: product3.Id, FromLocationId: locD.Id, ToLocationId: locE.Id, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = m3.RegisterEntry(pharmacymanager.RegisterEntryArgs{
		TenantId: testTenant, ProductId: product3.Id, LocationId: locE.Id, BatchId: rec3.BatchId, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if len(sink.facts) != 0 {
		t.Errorf("expected no facts, got %v", sink.facts)
	}

}
