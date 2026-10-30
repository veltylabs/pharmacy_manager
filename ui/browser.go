package ui

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	descPharmacy  lang.Text = "Manage pharmacy inventory, locations, stock movements and batches."
	descProducts  lang.Text = "Products catalogue, prices, active status and presentation."
	descLocations lang.Text = "Storage locations: warehouses, dispensing rooms and quarantine."
)

// Screens declara las pantallas que este módulo provee al menú de la app anfitriona.
// Ejemplo: screens := Screens(caller)
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{{
		ID:    ID,
		Label: "Pharmacy",
		Icon:  string(IconPharmacy),
		Help:  descPharmacy,
		Children: []view.Screen{
			{
				ID:         ID + ".products",
				Label:      "Products",
				Icon:       string(IconProducts),
				Permission: model.ResourceOf(&pharmacymanager.PharmacyProductModel),
				Presenter:  pharmacymanager.NewProductView(caller, view.WithDescription(descProducts)),
			},
			{
				ID:         ID + ".locations",
				Label:      "Locations",
				Icon:       string(IconLocations),
				Permission: model.ResourceOf(&pharmacymanager.PharmacyLocationModel),
				Presenter:  pharmacymanager.NewLocationView(caller, view.WithDescription(descLocations)),
			},
		},
	}}
}
