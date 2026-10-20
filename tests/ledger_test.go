//go:build !wasm

package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/stock"
	"webtyp.com/stock/stocktest"
)

// idPair joins an id chosen by the conformance suite ("prod1", "loc1", "lotA") to the id the module generated.
// A slice scanned linearly, never a map: no module code or test uses map[K]V.
type idPair struct{ Suite, DB string }

// ledger adapts a Module to stocktest.Ledger so that the shared stock rules run against this module.
type ledger struct {
	t         *testing.T
	m         *pharmacymanager.Module
	products  []idPair
	locations []idPair
	lots      []idPair // Suite = lot id, DB = batch id
	lotOf     []idPair // Suite = lot id, DB = product id of that lot
}

func dbID(pairs []idPair, suite string) (string, bool) {
	for _, p := range pairs {
		if p.Suite == suite {
			return p.DB, true
		}
	}
	return "", false
}

func suiteID(pairs []idPair, db string) string {
	for _, p := range pairs {
		if p.DB == db {
			return p.Suite
		}
	}
	return db
}

func (l *ledger) product(suite string) string {
	if id, ok := dbID(l.products, suite); ok {
		return id
	}
	p := newProduct(l.t, l.m, "sku-"+suite)
	l.products = append(l.products, idPair{Suite: suite, DB: p.Id})
	return p.Id
}

func (l *ledger) location(suite string) string {
	if id, ok := dbID(l.locations, suite); ok {
		return id
	}
	loc := newLocation(l.t, l.m, "loc "+suite)
	l.locations = append(l.locations, idPair{Suite: suite, DB: loc.Id})
	return loc.Id
}

// neverExpires stands for "does not expire" in this module: a pharmacy batch always has an expiry date (the plan's rule),
// so the suite's lot without expiry (0) is received with a date far beyond every other, which orders it last as the
// suite expects; Lots() translates it back to 0.
const neverExpires = int64(4102444800) // 2100-01-01 UTC

func (l *ledger) Receive(product, location, lot string, expiry int64, quantity float64) error {
	if expiry == 0 {
		expiry = neverExpires
	}
	mv, err := l.m.ReceiveStock(pharmacymanager.ReceiveStockArgs{
		TenantId: testTenant, ProductId: l.product(product), LocationId: l.location(location),
		BatchNumber: lot, ExpirationDate: expiry, Quantity: quantity,
	})
	if err != nil {
		return err
	}
	if _, ok := dbID(l.lots, lot); !ok {
		l.lots = append(l.lots, idPair{Suite: lot, DB: mv.BatchId})
		l.lotOf = append(l.lotOf, idPair{Suite: lot, DB: l.product(product)})
	}
	return nil
}

// mapErr translates the module's errors to the suite's: only "not enough stock" differs.
func mapErr(err error) error {
	if err == pharmacymanager.ErrInsufficientStock {
		return stock.ErrInsufficient
	}
	return err
}

func (l *ledger) Output(product, location string, m stock.Movement, quantity float64) ([]stock.Take, error) {
	mvs, err := l.m.RegisterOutput(pharmacymanager.RegisterOutputArgs{
		TenantId: testTenant, ProductId: l.product(product), LocationId: l.location(location),
		Quantity: quantity, MovementType: string(m),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	takes := make([]stock.Take, len(mvs))
	for i, mv := range mvs {
		takes[i] = stock.Take{LotID: suiteID(l.lots, mv.BatchId), Quantity: mv.Quantity}
	}
	return takes, nil
}

func (l *ledger) Transfer(product, from, to string, quantity float64) error {
	_, err := l.m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: testTenant, ProductId: l.product(product), FromLocationId: l.location(from), ToLocationId: l.location(to), Quantity: quantity,
	})
	return mapErr(err)
}

func (l *ledger) Lots(product, location string) ([]stock.Lot, error) {
	levels, err := l.m.ListStock(pharmacymanager.ListStockArgs{TenantId: testTenant, ProductId: l.product(product), LocationId: l.location(location)})
	if err != nil {
		return nil, err
	}
	var out []stock.Lot
	for _, level := range levels {
		batch, err := l.m.GetBatch(testTenant, level.BatchId)
		if err != nil {
			return nil, err
		}
		expiry := batch.ExpirationDate
		if expiry == neverExpires {
			expiry = 0
		}
		out = append(out, stock.Lot{ID: suiteID(l.lots, level.BatchId), Expiry: expiry, Quantity: level.Quantity})
	}
	return out, nil
}

func (l *ledger) Entries(product string) ([]stocktest.Entry, error) {
	mvs, err := l.m.ListMovements(pharmacymanager.ListMovementsArgs{TenantId: testTenant, ProductId: l.product(product), Limit: 1000})
	if err != nil {
		return nil, err
	}
	out := make([]stocktest.Entry, 0, len(mvs))
	for i := len(mvs) - 1; i >= 0; i-- { // the module lists newest first; the suite wants oldest first
		mv := mvs[i]
		out = append(out, stocktest.Entry{
			Type: stock.Movement(mv.MovementType), Location: suiteID(l.locations, mv.LocationId),
			LotID: suiteID(l.lots, mv.BatchId), Quantity: mv.Quantity,
		})
	}
	return out, nil
}

// TestStockConformance runs the shared stock rules (first expiry first out, expired lots unused, atomic failures,
// transfers, ledger balance, never negative) against this module.
func TestStockConformance(t *testing.T) {
	stocktest.Run(t, func(t *testing.T) stocktest.Ledger {
		return &ledger{t: t, m: newModule(t)}
	})
}
