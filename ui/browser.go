//go:build browser

package ui

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
)

// Browser builds the screen of the module for the app's modules/browser.go: the group "Farmacia" with its two children,
// Productos and Ubicaciones. There is no screen for stock, batches or movements in this version.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	productsView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".products",
		Presenter: pharmacymanager.NewProductView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &pharmacymanager.PharmacyProduct{
				TenantId:     tenantID,
				Unit:         pharmacymanager.UnitBox,
				TaxTreatment: pharmacymanager.TaxTaxed,
				IsActive:     true,
			}
		},
	})
	if err != nil {
		return nil, err
	}
	// The options of a closed list are set here, not in the model: ormc does not accept module constants inside
	// input.Select(...) in a model literal.
	productsView.SetOptions("unit",
		fmt.KeyValue{Key: pharmacymanager.UnitPiece, Value: "Piece"},
		fmt.KeyValue{Key: pharmacymanager.UnitBox, Value: "Box"},
		fmt.KeyValue{Key: pharmacymanager.UnitTablet, Value: "Tablet"},
		fmt.KeyValue{Key: pharmacymanager.UnitMl, Value: "Milliliter"},
		fmt.KeyValue{Key: pharmacymanager.UnitGram, Value: "Gram"},
	)
	productsView.SetOptions("tax_treatment",
		fmt.KeyValue{Key: pharmacymanager.TaxTaxed, Value: "Taxed"},
		fmt.KeyValue{Key: pharmacymanager.TaxExempt, Value: "Exempt"},
	)

	locationsView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".locations",
		Presenter: pharmacymanager.NewLocationView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &pharmacymanager.PharmacyLocation{TenantId: tenantID, IsActive: true}
		},
	})
	if err != nil {
		return nil, err
	}

	return platformd.NewGroupUIModule(ID, Label, IconPharmacy, []platformd.UIModule{
		platformd.NewUIModule(ID+".products", "Productos", IconProducts, productsView),
		platformd.NewUIModule(ID+".locations", "Ubicaciones", IconLocations, locationsView),
	}), nil
}
