package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func (p *PharmacyLocation) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	var levels []PharmacyStockLevel
	err := tx.ReadAll(func() model.Model { return &PharmacyStockLevel{} }, crud.Query{Conditions: []storage.Condition{storage.Eq("location_id", p.Id)}, Limit: 1}, func(mod model.Model) {
		levels = append(levels, *mod.(*PharmacyStockLevel))
	})
	if err != nil {
		return 0, err
	}
	if len(levels) > 0 {
		return 0, ErrLocationInUse
	}
	return crud.ApplyAndPublish, nil
}
