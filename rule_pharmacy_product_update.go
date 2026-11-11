package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/time"
)

// Update es la regla de negocio de actualizar pharmacy_product: old es la fila guardada y el receptor trae solo los campos
// enviados (fields). Revisa el producto resultante y, si cambió el SKU, que nadie más lo use.
func (next *PharmacyProduct) Update(a crud.CurrentUser, tx crud.Tx, old *PharmacyProduct, fields []string, m *Module) (crud.RuleAnswer, error) {
	merged := *old
	if err := applyProductFields(&merged, next, fields); err != nil {
		return 0, err
	}
	if err := checkProduct(&merged); err != nil {
		return 0, err
	}
	if merged.Sku != old.Sku {
		if err := checkSkuFree(tx, merged.Sku); err != nil {
			return 0, err
		}
	}
	merged.UpdatedAt = time.NowNano()
	next.Price = merged.Price // crud escribe next con las columnas de fields: si price viaja, debe ser el normalizado
	cols := []string{PharmacyProduct_.Price, PharmacyProduct_.UpdatedAt}
	if err := tx.UpdateFields(&merged, old.Id, cols); err != nil {
		return 0, err
	}
	return crud.ApplyAndPublish, nil
}
