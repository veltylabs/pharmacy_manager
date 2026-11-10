package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

func (p *LowStock) Read(a crud.CurrentUser, filter *LowStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	return &LowStockList{}, nil
}
