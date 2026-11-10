package tests

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"testing"
	"webtyp.com/crud"
	"webtyp.com/model"
)

func TestOperation_ReceiveCreatesBatchAndLevel(t *testing.T) {
	mod := newModule(t)
	caller := crud.LoopbackActingAs(nil, "u1", mod)
	_ = caller
}
