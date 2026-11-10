package seed

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/crud"
	"webtyp.com/fmt"
	"webtyp.com/time"
	"webtyp.com/view"
)

type Data struct {
	Locations []pharmacymanager.PharmacyLocation
	Products  []pharmacymanager.PharmacyProduct
}

const (
	locStore       = "Bodega"
	locCounter     = "Mostrador"
	expiryDays     = 365
	unitCostPesos  = 300
	pricePesos     = 2990
	demoQuantity   = 40
	counterPortion = 10
)

// Load is an internal seed helper.
func Load(m *pharmacymanager.Module, tenantID string) (Data, error) {
	// Omitting for brevity...
	return Data{}, nil
}

func Demo() view.Demo {
	return view.Demo{
		Label:      "Pharmacy",
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{},
		Migrations: nil,
		Screens:    func(d view.BrowserDeps) []view.Screen { return nil },
	}
}
