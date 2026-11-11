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
		Migrations: []ddl.Step{{Name: pharmacymanager.Definition.Name, Run: migrate.Migrate}},
		Screens: func(d view.BrowserDeps) []view.Screen {
			return ui.Screens(d.Caller)
		},
	}, nil
}
