package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/model"
)

func TestRowsDeclaredOnFields(t *testing.T) {
	prod := pharmacymanager.PharmacyProduct{
		Id:   "prod1",
		Name: "Paracetamol",
		Sku:  "MED-001",
	}
	rProd := model.RowOf(&prod)
	if rProd.ID != "prod1" || rProd.Title != "Paracetamol" || rProd.Detail != "MED-001" {
		t.Errorf("unexpected Product RowOf: %+v", rProd)
	}

	loc := pharmacymanager.PharmacyLocation{
		Id:   "loc1",
		Name: "Bodega Principal",
	}
	rLoc := model.RowOf(&loc)
	if rLoc.ID != "loc1" || rLoc.Title != "Bodega Principal" {
		t.Errorf("unexpected Location RowOf: %+v", rLoc)
	}
}
