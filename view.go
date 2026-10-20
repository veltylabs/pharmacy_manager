package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	titleProducts  = "Products"
	titleLocations = "Locations"
)

// Item projects a product as a view.Item: the label of a row and its secondary text.
func (m *PharmacyProduct) Item() view.Item {
	return view.Item{ID: m.Id, Label: m.Name, Description: m.Sku}
}

// Item projects a location as a view.Item.
func (m *PharmacyLocation) Item() view.Item {
	return view.Item{ID: m.Id, Label: m.Name}
}

// NewProductView builds the Presenter of the products: the engine a renderer (crudview or any other) wraps. The app picks
// the renderer; this module imports only view, model and router.
func NewProductView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListProducts, Save: OpUpsertProduct, Update: OpUpdateProduct, Delete: OpDeleteProduct},
		func() model.ModelSlice { return &PharmacyProductList{} })
	return view.New(b, &PharmacyProduct{}, view.WithTitle(titleProducts))
}

// NewLocationView builds the Presenter of the locations.
func NewLocationView(caller router.Caller) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListLocations, Save: OpUpsertLocation, Update: OpUpdateLocation, Delete: OpDeleteLocation},
		func() model.ModelSlice { return &PharmacyLocationList{} })
	return view.New(b, &PharmacyLocation{}, view.WithTitle(titleLocations))
}
