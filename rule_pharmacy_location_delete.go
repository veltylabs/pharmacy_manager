package pharmacymanager

import (
	"webtyp.com/crud"
)

// Delete es la regla de negocio de borrar pharmacy_location: una ubicación con stock no se borra.
func (p *PharmacyLocation) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkLocationDeletable(tx, p.Id); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
