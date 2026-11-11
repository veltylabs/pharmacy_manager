package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

// Read es la regla de negocio de leer low_stock: los productos activos en o por debajo de su mínimo.
func (p *LowStock) Read(a crud.CurrentUser, filter *LowStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.lowStockOn(m.store)
	if err != nil {
		return nil, err
	}
	list := LowStockList{}
	for i := range rows {
		list = append(list, &rows[i])
	}
	return &list, nil
}
