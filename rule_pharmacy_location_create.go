package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/time"
)

// Create es la regla de negocio de crear pharmacy_location: el nombre no puede repetirse.
func (p *PharmacyLocation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	if err := checkLocationNameFree(tx, p.Name); err != nil {
		return 0, err
	}
	p.UpdatedAt = time.NowNano()
	return crud.ApplyAndPublish, nil
}
