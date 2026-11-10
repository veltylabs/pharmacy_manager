package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func (next *PharmacyLocation) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyLocation, fields []string, m *Module) (crud.RuleAnswer, error) {
	if next.Name != old.Name {
		var proxy PharmacyLocation
		err := tx.ReadAll(func() model.Model { return &PharmacyLocation{} }, crud.Query{Conditions: []storage.Condition{storage.Eq("name", next.Name)}, Limit: 1}, func(mod model.Model) {
			proxy = *mod.(*PharmacyLocation)
		})
		if err != nil {
			return 0, err
		}
		if proxy.Id != "" {
			return 0, ErrLocationNameExists
		}
	}
	return crud.ApplyAndPublish, nil
}
