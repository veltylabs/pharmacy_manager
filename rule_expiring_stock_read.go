package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *ExpiringStock) Read(a crud.CurrentUser, filter *ExpiringStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	// Not fully implemented readers since we just want to compile
	return &ExpiringStockList{}, nil
}
