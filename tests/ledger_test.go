//go:build !wasm

package tests

import (
	"strings"
	"testing"
	"webtyp.com/model"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/stock"
	"webtyp.com/stock/stocktest"
	"webtyp.com/time"
)

// idPair une un id elegido por la suite de conformidad ("prod1", "loc1", "lotA") al id generado por el módulo.
// Un slice escaneado linealmente, nunca un map: ningún código o test de módulo usa map[K]V.
type idPair struct{ Suite, DB string }

// ledger adapta un Module a stocktest.Ledger para que las reglas de stock compartidas se ejecuten contra este módulo.
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

func (l *ledger) Receive(product, location, lot string, expiry time.Unix, quantity float64) error {
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

// mapErr traduce los errores del módulo a los de la suite: solo "no enough stock" difiere.
func mapErr(err error) error {
	if model.KindOf(err) == model.ErrorInvalid && strings.Contains(err.Error(), string(pharmacymanager.ErrInsufficientStock)) {
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

// TestStockConformance ejecuta las reglas de stock compartidas (primero en caducar primero en salir, lotes caducados sin usar, fallas atómicas,
// transferencias, balance del libro mayor, nunca negativo) contra este módulo.
func TestStockConformance(t *testing.T) {
	stocktest.Run(t, func(t *testing.T) stocktest.Ledger {
		return &ledger{t: t, m: newModule(t)}
	})
}
