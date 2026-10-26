//go:build browser

package ui

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/lang"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	descPharmacy  lang.Text = "Pharmacy management: catalog of medications, expiration dates and storage locations."
	descProducts  lang.Text = "Medications and pharmaceuticals catalog: presentation, active ingredients, unit and sale conditions."
	descLocations lang.Text = "Storage locations, shelves and warehouses for medications custody."
)

// Screens declara las pantallas que este módulo ofrece al menú de la app.
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{{
		ID:    ID,
		Label: Label,
		Icon:  string(IconPharmacy),
		Help:  descPharmacy,
		Children: []view.Screen{
			{
				ID:         ID + ".products",
				Label:      "Productos",
				Icon:       string(IconProducts),
				Permission: "pharmacy_product",
				Presenter:  pharmacymanager.NewProductView(caller, view.WithDescription(descProducts)),
			},
			{
				ID:         ID + ".locations",
				Label:      "Ubicaciones",
				Icon:       string(IconLocations),
				Permission: "pharmacy_location",
				Presenter:  pharmacymanager.NewLocationView(caller, view.WithDescription(descLocations)),
			},
		},
	}}
}
