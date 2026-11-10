package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func (p *PharmacyProduct) Delete(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	var batches []PharmacyBatch
	err := tx.ReadAll(func() model.Model { return &PharmacyBatch{} }, crud.Query{Conditions: []storage.Condition{storage.Eq("product_id", p.Id)}, Limit: 1}, func(mod model.Model) {
		batches = append(batches, *mod.(*PharmacyBatch))
	})
	if err != nil {
		return 0, err
	}
	if len(batches) > 0 {
		return 0, ErrProductInUse
	}
	return crud.ApplyAndPublish, nil
}
