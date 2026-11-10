package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/money"
	"webtyp.com/notify"
	"webtyp.com/orm"
	"webtyp.com/stock"
)

const TopicMovementRegistered = "pharmacy.movement.registered"
const sourceType = "pharmacy_movement"

const (
	msgNeedIDs      = "pharmacy: Deps.IDs is required"
	msgPublishCost  = "pharmacy: the stock change was saved but its cost fact could not be published:"
	msgBelowMinimum = "%v %s left; minimum %v."
	msgExpiring     = "%d expired batches and %d expiring within %d days."
)

type Deps struct {
	IDs        model.IDGenerator
	TenantID   string
	Publisher  events.Publisher
	MoneyScope string
	AlertArea  string
}

type Module struct {
	db        *orm.DB
	store     *orm.Store
	ids       model.IDGenerator
	tenantID  string
	pub       events.Publisher
	scope     string
	alertArea string
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err(msgNeedIDs)
	}
	return &Module{
		db:        db,
		store:     orm.TenantStore(db, deps.TenantID, deps.IDs),
		ids:       deps.IDs,
		tenantID:  deps.TenantID,
		pub:       deps.Publisher,
		scope:     deps.MoneyScope,
		alertArea: deps.AlertArea,
	}, nil
}

func checkProduct(p *PharmacyProduct) error {
	switch p.Unit {
	case UnitPiece, UnitBox, UnitTablet, UnitMl, UnitGram:
	default:
		return ErrUnit
	}
	switch p.TaxTreatment {
	case TaxTaxed, TaxExempt:
	default:
		return ErrTaxTreatmentRequired
	}
	if p.ForSale && p.Price <= 0 {
		return ErrSaleNeedsPrice
	}
	if !p.ForSale {
		p.Price = 0
	}
	return nil
}

func (m *Module) movementsFor(tenantId string, ids string) []PharmacyMovement {
	var mvs []PharmacyMovement
	return mvs
}

func (m *Module) publish(mv PharmacyMovement) {
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicMovementRegistered, Payload: &mv})
	}
}

func (m *Module) publishCosts(movements []PharmacyMovement, unitCosts []int64) error {
	if m.pub == nil || m.scope == "" {
		return nil
	}
	for i, mv := range movements {
		amount := int64(float64(unitCosts[i])*mv.Quantity + 0.5)
		if amount <= 0 {
			continue
		}
		price, err := money.New(amount, money.CLP)
		if err != nil {
			return fmt.Err(msgPublishCost, err)
		}
		fact := money.Fact{
			Kind:   money.Consumed,
			Side:   money.Outflow,
			Amount: price,
			Scope:  m.scope,
			Source: money.Ref{Type: sourceType, ID: mv.Id},
			At:     mv.CreatedAt.Unix(),
		}
		if err := money.Publish(m.pub, fact); err != nil {
			return fmt.Err(msgPublishCost, err)
		}
	}
	return nil
}
