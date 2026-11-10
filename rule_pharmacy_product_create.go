package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

func (p *PharmacyProduct) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkProduct(p); err != nil {
		return 0, err
	}
	var existing PharmacyProduct
	err := tx.ReadOne(&existing, p.Sku)
	if err == nil {
		return 0, ErrSkuExists
	}
	if !storage.IsNotFound(err) {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
