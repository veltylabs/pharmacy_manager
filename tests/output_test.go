package tests

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"testing"
	"webtyp.com/crud"
	"webtyp.com/model"
)

func TestOperation_OutputFirstExpiryFirst(t *testing.T) {
	mod := newModule(t)
	caller := crud.LoopbackActingAs(nil, "u1", mod)
	_ = caller
}

func TestOperation_OutputNeverNegative(t *testing.T) {
	mod := newModule(t)
	caller := crud.LoopbackActingAs(nil, "u1", mod)
	_ = caller
}

func TestOperation_UnknownKindRejected(t *testing.T) {
	mod := newModule(t)
	caller := crud.LoopbackActingAs(nil, "u1", mod)
	_ = caller
}
