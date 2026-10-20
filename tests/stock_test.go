package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/stock"
)

func TestTransferStock_MovesTheSameBatches(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	locA := newLocation(t, m, "A")
	locB := newLocation(t, m, "B")

	_, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, BatchNumber: "L1", Quantity: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	movs, err := m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: product.Id, FromLocationId: locA.Id, ToLocationId: locB.Id, Quantity: 4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movs) != 2 {
		t.Errorf("expected 2 movements, got %v", len(movs))
	}

	if movs[0].MovementType != string(stock.TransferOut) || movs[0].Quantity != 4 || movs[0].LocationId != locA.Id {
		t.Errorf("expected TransferOut 4 at A, got %v", movs[0])
	}
	if movs[1].MovementType != string(stock.TransferIn) || movs[1].Quantity != 4 || movs[1].LocationId != locB.Id {
		t.Errorf("expected TransferIn 4 at B, got %v", movs[1])
	}

	stockA, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, LocationId: locA.Id, ProductId: product.Id})
	stockB, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, LocationId: locB.Id, ProductId: product.Id})

	if len(stockA) != 1 || stockA[0].Quantity != 6 {
		t.Errorf("expected 6 in A, got %v", stockA)
	}
	if len(stockB) != 1 || stockB[0].Quantity != 4 {
		t.Errorf("expected 4 in B, got %v", stockB)
	}
	if stockA[0].BatchId != stockB[0].BatchId {
		t.Errorf("expected same BatchId, got %v and %v", stockA[0].BatchId, stockB[0].BatchId)
	}
}

func TestTransferStock_Rules(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	locA := newLocation(t, m, "A")
	locB := newLocation(t, m, "B")

	_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: locA.Id, BatchNumber: "L1", Quantity: 10,
	})

	_, err := m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: product.Id, FromLocationId: locA.Id, ToLocationId: locA.Id, Quantity: 4,
	})
	if err != pharmacymanager.ErrSameLocation {
		t.Errorf("expected ErrSameLocation, got %v", err)
	}

	_, err = m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: product.Id, FromLocationId: locA.Id, ToLocationId: locB.Id, Quantity: 11,
	})
	if err != pharmacymanager.ErrInsufficientStock {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}

	stockA, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, LocationId: locA.Id, ProductId: product.Id})
	stockB, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, LocationId: locB.Id, ProductId: product.Id})
	if len(stockA) != 1 || stockA[0].Quantity != 10 {
		t.Errorf("expected 10 in A, got %v", stockA)
	}
	if len(stockB) != 0 {
		t.Errorf("expected 0 in B, got %v", stockB)
	}
}

func TestRegisterEntry(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	rec, _ := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10,
	})

	mov, err := m.RegisterEntry(pharmacymanager.RegisterEntryArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchId: rec.BatchId, Quantity: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mov.MovementType != string(stock.AdjustIn) || mov.Quantity != 3 {
		t.Errorf("expected AdjustIn 3, got %v", mov)
	}

	stockL, _ := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, LocationId: loc.Id, ProductId: product.Id})
	if len(stockL) != 1 || stockL[0].Quantity != 13 {
		t.Errorf("expected level to be 13, got %v", stockL)
	}

	_, err = m.RegisterEntry(pharmacymanager.RegisterEntryArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchId: "unknown", Quantity: 3,
	})
	if err != pharmacymanager.ErrBatchNotFound {
		t.Errorf("expected ErrBatchNotFound, got %v", err)
	}

	_, err = m.RegisterEntry(pharmacymanager.RegisterEntryArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchId: rec.BatchId, Quantity: 0,
	})
	if _, ok := err.(pharmacymanager.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestReceiveStock_CostRules(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	_, err := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10, UnitCost: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10, UnitCost: 200,
	})
	if err != pharmacymanager.ErrBatchCostMismatch {
		t.Errorf("expected ErrBatchCostMismatch, got %v", err)
	}

	_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10, UnitCost: 0,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	batches, _ := m.ListBatches(testTenant, product.Id)
	if len(batches) != 1 || batches[0].UnitCost != 100 {
		t.Errorf("expected cost stays 100, got %v", batches)
	}
}

func TestListMovements(t *testing.T) {
	m := newModule(t)
	product := newProduct(t, m, "i1")
	loc := newLocation(t, m, "Loc")

	rec, _ := m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: neverExpires,
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, BatchNumber: "L1", Quantity: 10,
	})

	_, _ = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, Quantity: 2, MovementType: string(stock.InternalUse),
	})

	_, _ = m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: product.Id, LocationId: loc.Id, Quantity: 3, MovementType: string(stock.InternalUse),
	})

	movs, err := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: product.Id})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movs) != 3 {
		t.Errorf("expected 3 movements, got %v", len(movs))
	}
	if movs[0].Quantity != 3 { // newest first
		t.Errorf("expected newest first, got %v", movs[0])
	}

	movs2, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: product.Id, Limit: 2})
	if len(movs2) != 2 || movs2[0].Quantity != 3 || movs2[1].Quantity != 2 {
		t.Errorf("expected the 2 newest, got %v", movs2)
	}

	movs3, _ := m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: product.Id, BatchId: "other"})
	if len(movs3) != 0 {
		t.Errorf("expected none, got %v", len(movs3))
	}
	_ = rec
}
