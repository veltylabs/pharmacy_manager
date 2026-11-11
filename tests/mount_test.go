package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/crud"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/unixid"
)

func TestMount_MatchesDefinition(t *testing.T) {
	db := orm.New(mem.New())
	ids, _ := unixid.NewUnixID()
	m, _ := pharmacymanager.New(db, pharmacymanager.Deps{TenantID: testTenant, IDs: ids})

	if err := crud.Mount(nil, nil, nil, m); err != nil {
		t.Fatal(err)
	}
}
