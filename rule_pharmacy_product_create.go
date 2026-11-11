package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/time"
)

// Create es la regla de negocio de crear pharmacy_product: unidad, tratamiento y precio válidos, y un SKU que nadie usa.
func (p *PharmacyProduct) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkProduct(p); err != nil {
		return 0, err
	}
	if err := checkSkuFree(tx, p.Sku); err != nil {
		return 0, err
	}
	p.UpdatedAt = time.NowNano()
	return crud.ApplyAndPublish, nil
}
