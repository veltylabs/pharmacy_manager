← [Stage 5](PLAN_STAGE_5_METHODS.md) | Next → [Stage 7](PLAN_STAGE_7_TESTS_DOCS.md)

# Stage 6 — `view.go`, `ui/`, demo

This module has NO `module/` package: do not create one. `ui/browser.go` and `ui/module.go` already have the right shape (`Screens(caller router.Caller)`, `ID`, `Label`, the icons): **do not change them**. If `view.NewCallerLister` or `view.Ops` differ from this stage in `view@v0.6.74`, run `go doc webtyp.com/view Ops` and name the difference in the PR; the reference is https://github.com/veltylabs/device_manager/blob/main/view.go.

## 6.1 `view.go` — replace the whole file with exactly

`view.Ops` has only `Module`, `Actions` and `Page` (the old `List`, `Save`, `Update`, `Delete` fields are gone): the operations are derived from the module and the entity, and the model of each list is `model.TableSlice`.

```go
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
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyProductList{} })
	allOpts := append([]view.Option{view.WithTitle(titleProducts)}, opts...)
	return view.New(b, &PharmacyProduct{}, allOpts...)
}

// NewLocationView construye el Presenter de las ubicaciones.
//
// Ejemplo: p := NewLocationView(caller)
func NewLocationView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(view.NewRecords().Caller(caller),
		view.Ops{Module: ModelName, Page: view.Page{First: view.All}},
		func() model.TableSlice { return &PharmacyLocationList{} })
	allOpts := append([]view.Option{view.WithTitle(titleLocations)}, opts...)
	return view.New(b, &PharmacyLocation{}, allOpts...)
}
```

## 6.2 `seed/demo.go` (new, `//go:build wasm`), written exactly; then `git rm web/browser.go` (`test ! -e web`)

`seed/seed.go` (`Load`) does not change: it uses the module's own methods, which are kept. Write:

```go
//go:build wasm

package seed

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"github.com/veltylabs/pharmacy_manager/migrate"
	"github.com/veltylabs/pharmacy_manager/ui"
	"webtyp.com/crud"
	"webtyp.com/ddl"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
	"webtyp.com/view"
)

const demoTenantID = "demo"

// Demo crea una demostración ejecutable del módulo pharmacy_manager sobre una base en memoria con datos semilla.
//
// Ejemplo: demo, err := seed.Demo()
func Demo() (view.Demo, error) {
	db := orm.New(mem.New())
	ids, err := unixid.NewUnixID()
	if err != nil {
		return view.Demo{}, err
	}
	mod, err := pharmacymanager.New(db, pharmacymanager.Deps{IDs: ids, TenantID: demoTenantID})
	if err != nil {
		return view.Demo{}, err
	}
	if _, err := Load(mod, demoTenantID); err != nil {
		return view.Demo{}, err
	}
	return view.Demo{
		Label:      ui.Label,
		Timezone:   "America/Santiago",
		ActingAs:   "u1",
		Modules:    []crud.Module{mod},
		Migrations: []ddl.Step{{Name: pharmacymanager.ModelName, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller)
		},
	}, nil
}
```
If `go get webtyp.com/unixid@latest` is needed, run it and `go mod tidy`. The `wasm` destination of `gotest` compiles `seed/demo.go`; do not run the demo (headless CI). `grep -rn "webtyp.com/layout\|router/loopback\|events/mock" --include=*.go . | grep -v _test.go` → empty. Commit and push.
