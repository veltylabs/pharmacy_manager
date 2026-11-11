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

// NewProductView construye el Presenter de los productos: el motor que un renderizador (crudview u otro) envuelve. La app
// elige el renderizador; este módulo solo importa view, model y router.
//
// Ejemplo: p := NewProductView(caller)
func NewProductView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: Definition.Name, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyProductList{} })
	allOpts := append([]view.Option{view.WithTitle(titleProducts)}, opts...)
	return view.New(b, &PharmacyProduct{}, allOpts...)
}

// NewLocationView construye el Presenter de las ubicaciones.
//
// Ejemplo: p := NewLocationView(caller)
func NewLocationView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: Definition.Name, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyLocationList{} })
	allOpts := append([]view.Option{view.WithTitle(titleLocations)}, opts...)
	return view.New(b, &PharmacyLocation{}, allOpts...)
}
