package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

func (next *PharmacyProduct) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyProduct, fields []string, m *Module) (crud.RuleAnswer, error) {
	if err := checkProduct(next); err != nil {
		return 0, err
	}
	if next.Sku != old.Sku {
		var existing PharmacyProduct
		err := tx.ReadOne(&existing, next.Sku)
		if err == nil {
			return 0, ErrSkuExists
		}
		if !storage.IsNotFound(err) {
			return 0, err
		}
	}
	return crud.ApplyAndPublish, nil
}
