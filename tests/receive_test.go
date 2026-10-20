package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/stock"
	"webtyp.com/time"
)

// receive builds the arguments of a ReceiveStock for a product and a location; the tests change what they exercise.
func receive(productId, locationId, batch string, expiry int64, quantity float64) pharmacymanager.ReceiveStockArgs {
	return pharmacymanager.ReceiveStockArgs{
		TenantId: testTenant, ProductId: productId, LocationId: locationId, BatchNumber: batch, ExpirationDate: expiry, Quantity: quantity,
	}
}

func TestReceiveStock_AddsUpAndWritesTheLedger(t *testing.T) {
	m := newModule(t)
	p, loc := newProduct(t, m, "p1"), newLocation(t, m, "Store")
	future := time.Now() + 30*stock.SecondsPerDay

	first, err := m.ReceiveStock(receive(p.Id, loc.Id, "L1", future, 10))
	if err != nil {
		t.Fatalf("first receive: %v", err)
	}
	if _, err := m.ReceiveStock(receive(p.Id, loc.Id, "L1", future, 5)); err != nil {
		t.Fatalf("second receive into the same batch: %v", err)
	}

	levels, err := m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: p.Id})
	if err != nil || len(levels) != 1 || levels[0].Quantity != 15 {
		t.Fatalf("one level of 15 expected, got %v, %v", levels, err)
	}
	if first.MovementType != string(stock.Purchase) || first.Quantity != 10 {
		t.Errorf("the first movement must be a PURCHASE of 10, got %+v", first)
	}
}

func TestReceiveStock_Rules(t *testing.T) {
	m := newModule(t)
	p, loc := newProduct(t, m, "p1"), newLocation(t, m, "Store")
	future := time.Now() + 30*stock.SecondsPerDay

	cases := []struct {
		name string
		args pharmacymanager.ReceiveStockArgs
		want error
	}{
		{"no expiry date", receive(p.Id, loc.Id, "L1", 0, 1), pharmacymanager.ErrExpiryRequired},
		{"no batch number", receive(p.Id, loc.Id, "", future, 1), pharmacymanager.ErrBatchNumber},
		{"unknown product", receive("nope", loc.Id, "L1", future, 1), pharmacymanager.ErrProductNotFound},
		{"unknown location", receive(p.Id, "nope", "L1", future, 1), pharmacymanager.ErrLocationNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := m.ReceiveStock(c.args); err != c.want {
				t.Errorf("got %v, want %v", err, c.want)
			}
		})
	}

	if _, err := m.ReceiveStock(receive(p.Id, loc.Id, "L9", future, 3)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReceiveStock(receive(p.Id, loc.Id, "L9", future+1, 3)); err != pharmacymanager.ErrBatchExpiryMismatch {
		t.Errorf("the same batch with another expiry: got %v, want ErrBatchExpiryMismatch", err)
	}
}
