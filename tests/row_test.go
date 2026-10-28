package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
)

func TestItemsDeclaredOnFields(t *testing.T) {
	prod := pharmacymanager.PharmacyProduct{
		Id:   "prod1",
		Name: "Paracetamol",
		Sku:  "MED-001",
	}
	rProd := prod.Item()
	if rProd.ID != "prod1" || rProd.Label != "Paracetamol" || rProd.Description != "MED-001" {
		t.Errorf("unexpected Product Item(): %+v", rProd)
	}

	loc := pharmacymanager.PharmacyLocation{
		Id:   "loc1",
		Name: "Bodega Principal",
	}
	rLoc := loc.Item()
	if rLoc.ID != "loc1" || rLoc.Label != "Bodega Principal" {
		t.Errorf("unexpected Location Item(): %+v", rLoc)
	}
}
