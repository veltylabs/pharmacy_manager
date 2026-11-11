package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/time"
)

// Create es la regla de negocio de crear pharmacy_operation: ejecuta la operación de stock (receive, entry, output o
// transfer) dentro de la transacción y deja en la fila quién la hizo y los ids de los movimientos que produjo. Si el
// hecho de costo o la alerta de una salida no se pueden publicar, la operación completa se deshace.
func (p *PharmacyOperation) Create(a crud.CurrentUser, tx crud.Tx, m *Module) (crud.RuleAnswer, error) {
	eff, err := m.runOperation(tx, p, a.UserID)
	if err != nil {
		return 0, err
	}
	p.ActorId = a.UserID
	p.MovementIds = movementIds(eff.movements)
	p.CreatedAt = time.NowNano()
	if err := m.announce(eff); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
