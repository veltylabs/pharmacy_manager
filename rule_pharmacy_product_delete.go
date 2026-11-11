package pharmacymanager

import (
	"webtyp.com/crud"
)

// Delete es la regla de negocio de borrar pharmacy_product: un producto con lotes no se borra. El receptor es la fila guardada.
func (p *PharmacyProduct) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkProductDeletable(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
