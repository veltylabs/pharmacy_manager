package pharmacymanager

import (
	"webtyp.com/msgtype"
	"webtyp.com/notify"
)

const (
	AlertBelowMinimum = "pharmacy.below_minimum"
	AlertExpiring     = "pharmacy.expiring"
)

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

func AlertKinds(area string) []notify.Kind {
	return []notify.Kind{
		{Code: AlertBelowMinimum, Name: textBelowMinimumName, Help: textBelowMinimumHelp, Area: area},
		{Code: AlertExpiring, Name: textExpiringName, Help: textExpiringHelp, Area: area},
	}
}

func (m *Module) raise(a notify.Alert) error {
	if m.pub == nil || m.alertArea == "" {
		return nil
	}
	a.Scope, a.Level = m.alertArea, msgtype.Warning
	return notify.Raise(m.pub, a)
}

func (m *Module) totalProduct(tenantId, id string) (float64, error) {
	return 0, nil
}
