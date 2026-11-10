package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func (p *PharmacyLocation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	var proxy PharmacyLocation
	err := tx.ReadAll(func() model.Model { return &PharmacyLocation{} }, crud.Query{Conditions: []storage.Condition{storage.Eq("name", p.Name)}, Limit: 1}, func(mod model.Model) {
		proxy = *mod.(*PharmacyLocation)
	})
	if err != nil {
		return 0, err
	}
	if proxy.Id != "" {
		return 0, ErrLocationNameExists
	}
	return crud.ApplyAndPublish, nil
}
