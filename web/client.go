//go:build wasm

package main

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"github.com/veltylabs/pharmacy_manager/seed"
	"github.com/veltylabs/pharmacy_manager/ui"

	. "webtyp.com/dom"
	"webtyp.com/events/mock"
	. "webtyp.com/html"
	"webtyp.com/layout/platformd"
	"webtyp.com/orm"
	"webtyp.com/router/loopback"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

// demoTenantID is the only tenant of this in-browser demo.
const demoTenantID = "demo"

// demoUser is the fixed identity the demo shell shows: the demo has no login.
type demoUser struct{}

func (demoUser) UserName() string    { return "Demo" }
func (demoUser) UserAvatar() string  { return "" }
func (demoUser) UserRoles() []string { return []string{"Administrador"} }

func main() {
	ids, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}
	db := orm.New(mem.New())
	broker := &mock.Broker{}

	pm, err := pharmacymanager.New(db, pharmacymanager.Deps{IDs: ids, Publisher: broker})
	if err != nil {
		panic(err)
	}
	if _, err := seed.Load(pm, demoTenantID); err != nil {
		panic(err)
	}

	caller := loopback.WithTenant(demoTenantID, pm)
	screen, err := ui.Browser(caller, ids, demoTenantID)
	if err != nil {
		panic(err)
	}

	homeMod := platformd.NewUIModule("home", "Inicio", ui.IconPharmacy, Div().Text("Vista de Inicio (Dummy)"))
	settingsMod := platformd.NewUIModule("settings", "Ajustes", ui.IconPharmacy, Div().Text("Vista de Ajustes (Dummy)"))

	p := &platformd.Platform{
		AppName:   "WebTyp App Demo",
		User:      demoUser{},
		Modules:   []platformd.UIModule{homeMod, screen, settingsMod},
		DefaultID: ui.ID,
	}
	Append("body", p)
	select {}
}
