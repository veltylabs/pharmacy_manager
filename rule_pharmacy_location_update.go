package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/time"
)

// Update es la regla de negocio de actualizar pharmacy_location: si cambia el nombre, nadie más puede tenerlo.
func (next *PharmacyLocation) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyLocation, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	if err := applyLocationFields(&merged, next, fields); err != nil {
		return 0, err
	}
	if merged.Name != old.Name {
		if err := checkLocationNameFree(tx, merged.Name); err != nil {
			return 0, err
		}
	}
	merged.UpdatedAt = time.NowNano()
	if err := tx.UpdateFields(&merged, old.Id, []string{PharmacyLocation_.UpdatedAt}); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
