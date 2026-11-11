package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
)

// Read es la regla de negocio de leer expiring_stock: el stock que caduca dentro de filter.WithinDays días.
func (p *ExpiringStock) Read(a crud.CurrentUser, filter *ExpiringStock, args crud.ReadArgs, m *Module) (model.ModelSlice, error) {
	rows, err := m.expiringOn(m.store, filter.WithinDays)
	if err != nil {
		return nil, err
	}
	list := ExpiringStockList{}
	for i := range rows {
		list = append(list, &rows[i])
	}
	return &list, nil
}
