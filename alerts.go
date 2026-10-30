package pharmacymanager

import (
	"webtyp.com/msgtype"
	"webtyp.com/notify"
)

// Códigos de alerta que este módulo lanza (notify.Alert.Code).
const (
	AlertBelowMinimum = "pharmacy.below_minimum"
	AlertExpiring     = "pharmacy.expiring"
)

// Tipos de origen de las alertas (notify.Ref.Type).
const (
	alertSourceProduct = "pharmacy_product"
	alertSourceReview  = "pharmacy_expiry_review"
)

const (
	textBelowMinimumName = "Pharmacy below minimum"
	textBelowMinimumHelp = "An product reached its minimum quantity after an output. Restock it."
	textExpiringName     = "Pharmacy expiring"
	textExpiringHelp     = "The daily review found batches with stock that are expired or expire soon."
)

// AlertKinds es el catálogo de alertas que lanza este módulo, para el notify.Registry de la aplicación.
// area es el código notify.Area de la aplicación (el mismo valor que Deps.AlertArea).
func AlertKinds(area string) []notify.Kind {
	return []notify.Kind{
		{Code: AlertBelowMinimum, Name: textBelowMinimumName, Help: textBelowMinimumHelp, Area: area},
		{Code: AlertExpiring, Name: textExpiringName, Help: textExpiringHelp, Area: area},
	}
}

// raise publica cuando las alertas están activadas; si no (sin Publisher o sin AlertArea) no hace nada.
func (m *Module) raise(a notify.Alert) error {
	if m.pub == nil || m.alertArea == "" {
		return nil
	}
	a.Scope, a.Level = m.alertArea, msgtype.Warning
	return notify.Raise(m.pub, a)
}

// totalProduct es el stock del producto en cada ubicación y lote.
func (m *Module) totalProduct(tenantId, id string) (float64, error) {
	levels, err := m.ListStock(ListStockArgs{TenantId: tenantId, ProductId: id})
	if err != nil {
		return 0, err
	}
	var total float64
	for _, l := range levels {
		total += l.Quantity
	}
	return total, nil
}
