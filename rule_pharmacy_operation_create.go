package pharmacymanager

import "webtyp.com/crud"

func (p *PharmacyOperation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	return crud.ApplyAndPublish, nil
}
