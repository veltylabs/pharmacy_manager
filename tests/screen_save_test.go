package tests

import (
	"testing"
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"github.com/veltylabs/pharmacy_manager/ui"
	"webtyp.com/model"
	"webtyp.com/router/loopback"
	"webtyp.com/view"
)

func TestScreenSaveAndDelete(t *testing.T) {
	m := newModule(t)
	caller := view.NewRecords().Caller(loopback.ActingAs(testTenant, "u1", m))
	screens := ui.Screens(caller)
	if len(screens) == 0 || len(screens[0].Children) < 2 {
		t.Fatal("expected screens with children")
	}

	// Products screen
	prodSaver := screens[0].Children[0].Presenter.(view.Saver)
	prodDeleter := screens[0].Children[0].Presenter.(view.Deleter)

	p := &pharmacymanager.PharmacyProduct{
		TenantId:     testTenant,
		Sku:          "p-screen-1",
		Name:         "Aspirin 500mg",
		Unit:         pharmacymanager.UnitBox,
		TaxTreatment: pharmacymanager.TaxTaxed,
		ForSale:      true,
		Price:        2000,
		IsActive:     true,
	}

	var saveErr error
	prodSaver.Save([]model.Model{p}, func(err error) { saveErr = err })
	if saveErr != nil {
		t.Fatalf("product save failed: %v", saveErr)
	}

	// Duplicate product conflict
	pDup := &pharmacymanager.PharmacyProduct{
		TenantId:     testTenant,
		Sku:          "p-screen-1",
		Name:         "Aspirin Duplicate",
		Unit:         pharmacymanager.UnitBox,
		TaxTreatment: pharmacymanager.TaxTaxed,
		ForSale:      true,
		Price:        2000,
		IsActive:     true,
	}
	prodSaver.Save([]model.Model{pDup}, func(err error) { saveErr = err })
	if model.KindOf(saveErr) != model.ErrorConflict {
		t.Fatalf("expected conflict on duplicate sku, got %v", saveErr)
	}

	// Delete product
	items := screens[0].Children[0].Presenter.Items()
	if len(items) == 0 {
		t.Fatal("expected items in products screen")
	}
	var delErr error
	prodDeleter.Delete([]string{items[0].ID}, func(err error) { delErr = err })
	if delErr != nil {
		t.Fatalf("product delete failed: %v", delErr)
	}

	// Locations screen
	locSaver := screens[0].Children[1].Presenter.(view.Saver)
	locDeleter := screens[0].Children[1].Presenter.(view.Deleter)

	loc := &pharmacymanager.PharmacyLocation{
		TenantId: testTenant,
		Name:     "Central Warehouse",
		IsActive: true,
	}
	locSaver.Save([]model.Model{loc}, func(err error) { saveErr = err })
	if saveErr != nil {
		t.Fatalf("location save failed: %v", saveErr)
	}

	// Duplicate location conflict
	locDup := &pharmacymanager.PharmacyLocation{
		TenantId: testTenant,
		Name:     "Central Warehouse",
		IsActive: true,
	}
	locSaver.Save([]model.Model{locDup}, func(err error) { saveErr = err })
	if model.KindOf(saveErr) != model.ErrorConflict {
		t.Fatalf("expected conflict on duplicate location name, got %v", saveErr)
	}

	locItems := screens[0].Children[1].Presenter.Items()
	if len(locItems) == 0 {
		t.Fatal("expected items in locations screen")
	}
	locDeleter.Delete([]string{locItems[0].ID}, func(err error) { delErr = err })
	if delErr != nil {
		t.Fatalf("location delete failed: %v", delErr)
	}
}
