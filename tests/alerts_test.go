package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/events"
	"webtyp.com/events/mock"
	"webtyp.com/notify"
	"webtyp.com/orm"
	"webtyp.com/stock"
	"webtyp.com/storage/mem"
	"webtyp.com/time"
)

type recordingPublisher struct {
	events []events.Event
}

func (p *recordingPublisher) Publish(e events.Event) {
	p.events = append(p.events, e)
}

func (p *recordingPublisher) Subscribe(topic string, handler func(events.Event)) {
}

func TestAlerts_AlertKinds(t *testing.T) {
	kinds := pharmacymanager.AlertKinds("supplies")
	if len(kinds) != 2 {
		t.Fatalf("expected 2 kinds, got %d", len(kinds))
	}
	if kinds[0].Code != pharmacymanager.AlertBelowMinimum || kinds[0].Area != "supplies" {
		t.Errorf("unexpected kind 0: %+v", kinds[0])
	}
	if kinds[1].Code != pharmacymanager.AlertExpiring || kinds[1].Area != "supplies" {
		t.Errorf("unexpected kind 1: %+v", kinds[1])
	}
}

func TestAlerts_RegisterOutput(t *testing.T) {
	cases := []struct {
		name       string
		total      float64
		min        float64
		output     float64
		alertArea  string
		publisher  bool
		wantAlert  bool
		wantBody   string
		isTransfer bool
	}{
		{"total 12, alerts on, output 3", 12, 10, 3, "pharmacy", true, true, "9 BOX left; minimum 10.", false},
		{"total 12, output 2 (lands on 10)", 12, 10, 2, "pharmacy", true, true, "10 BOX left; minimum 10.", false},
		{"total 9, output 1", 9, 10, 1, "pharmacy", true, false, "", false},
		{"total 20, output 5", 20, 10, 5, "pharmacy", true, false, "", false},
		{"min 0, any output", 12, 0, 3, "pharmacy", true, false, "", false},
		{"AlertArea == '', output 3 from 12", 12, 10, 3, "", true, false, "", false},
		{"Publisher == nil, output 3 from 12", 12, 10, 3, "pharmacy", false, false, "", false},
		{"transfer between two locations", 12, 10, 3, "pharmacy", true, false, "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var pub events.Publisher
			var broker *mock.Broker
			if c.publisher {
				broker = &mock.Broker{}
				pub = broker
			}

			m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{
				IDs:       &testIDGen{},
				Publisher: pub,
				AlertArea: c.alertArea,
			})
			if err != nil {
				t.Fatal(err)
			}

			var recorded []notify.Alert
			if broker != nil {
				broker.Subscribe(notify.TopicAlert, func(e events.Event) {
					recorded = append(recorded, *e.Payload.(*notify.Alert))
				})
			}

			p, err := m.CreateProduct(pharmacymanager.PharmacyProduct{
				TenantId: testTenant, Sku: "sku", Name: "Prod", Unit: pharmacymanager.UnitBox, MinQuantity: c.min, IsActive: true, ForSale: true, TaxTreatment: pharmacymanager.TaxTaxed, Price: 1000,
			})
			if err != nil {
				t.Fatal(err)
			}

			loc1 := newLocation(t, m, "Loc1")
			loc2 := newLocation(t, m, "Loc2")

			if c.total > 0 {
				_, err = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{
					ExpirationDate: time.NowUnix() + 100*time.Day,
					TenantId:       testTenant,
					ProductId:      p.Id,
					LocationId:     loc1.Id,
					BatchNumber:    "B1",
					Quantity:       c.total,
				})
				if err != nil {
					t.Fatal(err)
				}
			}

			if c.isTransfer {
				_, err = m.TransferStock(pharmacymanager.TransferStockArgs{
					TenantId:       testTenant,
					ProductId:      p.Id,
					FromLocationId: loc1.Id,
					ToLocationId:   loc2.Id,
					Quantity:       c.output,
				})
			} else {
				_, err = m.RegisterOutput(output(p.Id, loc1.Id, stock.Sale, c.output))
			}
			if err != nil {
				t.Fatalf("output error: %v", err)
			}

			if c.wantAlert {
				if len(recorded) != 1 {
					t.Fatalf("expected 1 alert, got %d", len(recorded))
				}
				a := recorded[0]
				if a.Code != pharmacymanager.AlertBelowMinimum {
					t.Errorf("wrong code: %q", a.Code)
				}
				if a.Scope != c.alertArea {
					t.Errorf("wrong scope: %q", a.Scope)
				}
				if a.Source.Type != "pharmacy_product" || a.Source.ID != p.Id {
					t.Errorf("wrong source: %+v", a.Source)
				}
				if a.Body != c.wantBody {
					t.Errorf("wrong body: got %q, want %q", a.Body, c.wantBody)
				}
			} else {
				if len(recorded) != 0 {
					t.Fatalf("expected no alerts, got %d", len(recorded))
				}
			}
		})
	}
}

func TestAlerts_ReviewExpiring(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(t *testing.T, m *pharmacymanager.Module, pID, lID string)
		withinDays int
		alertArea  string
		wantAlert  bool
		wantBody   string
		wantRows   int
		wantError  error
	}{
		{
			"one batch expired, one expiring in 5 days, one in 60",
			func(t *testing.T, m *pharmacymanager.Module, pID, lID string) {
				now := time.NowUnix()
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now - 1*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B1", Quantity: 5})
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now + 5*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B2", Quantity: 5})
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now + 60*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B3", Quantity: 5})
			},
			30, "pharmacy", true, "1 expired batches and 1 expiring within 30 days.", 2, nil,
		},
		{
			"nothing expiring",
			func(t *testing.T, m *pharmacymanager.Module, pID, lID string) {
				now := time.NowUnix()
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now + 60*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B3", Quantity: 5})
			},
			30, "pharmacy", false, "", 0, nil,
		},
		{
			"alerts off",
			func(t *testing.T, m *pharmacymanager.Module, pID, lID string) {
				now := time.NowUnix()
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now - 1*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B1", Quantity: 5})
				_, _ = m.ReceiveStock(pharmacymanager.ReceiveStockArgs{ExpirationDate: now + 5*time.Day, TenantId: testTenant, ProductId: pID, LocationId: lID, BatchNumber: "B2", Quantity: 5})
			},
			30, "", false, "", 2, nil,
		},
		{
			"withinDays < 0",
			func(t *testing.T, m *pharmacymanager.Module, pID, lID string) {},
			-1, "pharmacy", false, "", 0, pharmacymanager.ErrDays,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			broker := &mock.Broker{}
			m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{
				IDs:       &testIDGen{},
				Publisher: broker,
				AlertArea: c.alertArea,
			})
			if err != nil {
				t.Fatal(err)
			}

			var recorded []notify.Alert
			broker.Subscribe(notify.TopicAlert, func(e events.Event) {
				recorded = append(recorded, *e.Payload.(*notify.Alert))
			})

			p := newProduct(t, m, "p")
			l := newLocation(t, m, "l")
			c.setup(t, m, p.Id, l.Id)

			rows, err := m.ReviewExpiring(testTenant, c.withinDays)
			if err != c.wantError {
				t.Fatalf("error: got %v, want %v", err, c.wantError)
			}
			if len(rows) != c.wantRows {
				t.Errorf("rows: got %d, want %d", len(rows), c.wantRows)
			}
			if c.wantAlert {
				if len(recorded) != 1 {
					t.Fatalf("expected 1 alert, got %d", len(recorded))
				}
				a := recorded[0]
				if a.Code != pharmacymanager.AlertExpiring {
					t.Errorf("wrong code: %q", a.Code)
				}
				if a.Scope != c.alertArea {
					t.Errorf("wrong scope: %q", a.Scope)
				}
				if a.Source.Type != "pharmacy_expiry_review" {
					t.Errorf("wrong source: %+v", a.Source)
				}
				if a.Body != c.wantBody {
					t.Errorf("wrong body: got %q, want %q", a.Body, c.wantBody)
				}
			} else {
				if len(recorded) != 0 {
					t.Fatalf("expected no alerts, got %d", len(recorded))
				}
			}
		})
	}
}
