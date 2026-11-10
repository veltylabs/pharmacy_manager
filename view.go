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

func NewProductView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyProductList{} })
	allOpts := append([]view.Option{view.WithTitle(titleProducts)}, opts...)
	return view.New(b, &PharmacyProduct{}, allOpts...)
}

func NewLocationView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyLocationList{} })
	allOpts := append([]view.Option{view.WithTitle(titleLocations)}, opts...)
	return view.New(b, &PharmacyLocation{}, allOpts...)
}
