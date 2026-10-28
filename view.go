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

// NewProductView builds the Presenter of the products: the engine a renderer (crudview or any other) wraps. The app picks
// the renderer; this module imports only view, model and router.
func NewProductView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, List: OpListProducts, Save: OpUpsertProduct, Update: OpUpdateProduct, Delete: OpDeleteProduct},
		func() model.ModelSlice { return &PharmacyProductList{} })
	allOpts := append([]view.Option{view.WithTitle(titleProducts)}, opts...)
	return view.New(b, &PharmacyProduct{}, allOpts...)
}

// NewLocationView builds the Presenter of the locations.
func NewLocationView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, List: OpListLocations, Save: OpUpsertLocation, Update: OpUpdateLocation, Delete: OpDeleteLocation},
		func() model.ModelSlice { return &PharmacyLocationList{} })
	allOpts := append([]view.Option{view.WithTitle(titleLocations)}, opts...)
	return view.New(b, &PharmacyLocation{}, allOpts...)
}
