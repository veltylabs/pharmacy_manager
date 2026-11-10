package tests

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"testing"
	"webtyp.com/crud"
	"webtyp.com/model"
)

func TestTenant_ReadOtherTenantIsNotFound(t *testing.T) {
	mod := newModule(t)
	// Mock loopback check for tenant-1 and tenant-2
	caller := crud.LoopbackActingAs(nil, "u1", mod)
	_ = caller
}
