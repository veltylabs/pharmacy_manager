package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/money"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// TopicMovementRegistered se publica una vez por movimiento de stock, después de que la transacción se confirma. Payload: *PharmacyMovement.
const TopicMovementRegistered = "pharmacy.movement.registered"

const (
	// sourceType es el Ref.Type de los hechos de costo: el movimiento es su causa.
	sourceType = "pharmacy_movement"

	msgNeedIDs      = "pharmacy: Deps.IDs is required"
	msgNeedTenant   = "pharmacy: Deps.TenantID is required"
	msgPublishCost  = "pharmacy: the stock change was saved but its cost fact could not be published:"
	msgBelowMinimum = "%v %s left; minimum %v."
	msgExpiring     = "%d expired batches and %d expiring within %d days."
)

// Deps son lo que el módulo necesita de la aplicación que lo compone.
type Deps struct {
	IDs        model.IDGenerator // required: the module never builds a generator
	TenantID   string            // optional: the installation's tenant
	Publisher  events.Publisher  // optional: nil disables the events and the cost facts
	MoneyScope string            // optional: the area of the business the cost facts belong to; "" disables the cost facts
	AlertArea  string            // optional: the application's notify.Area code for this module's alerts; "" disables them
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

// Ejemplo: pm, err := New(db, deps)
func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.TenantID == "" {
		return nil, fmt.Err(msgNeedTenant)
	}
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

// publishCosts anuncia lo que un retiro consumió, al costo: un hecho money.Consumed por movimiento cuyo lote tiene costo unitario.
// No hace nada a menos que la aplicación diera un Publisher y un MoneyScope.
// productColumns son las columnas que cambian al actualizar un producto completo (todas menos id y tenant_id).
var productColumns = []string{PharmacyProduct_.Sku, PharmacyProduct_.Name, PharmacyProduct_.Unit, PharmacyProduct_.TaxTreatment,
	PharmacyProduct_.ForSale, PharmacyProduct_.Price, PharmacyProduct_.Controlled, PharmacyProduct_.Barcode,
	PharmacyProduct_.RegistryNumber, PharmacyProduct_.MinQuantity, PharmacyProduct_.IsActive, PharmacyProduct_.UpdatedAt}

func (m *Module) CreateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	if p.Id == "" {
		p.Id = m.ids.NewID()
	}
	p.UpdatedAt = time.NowNano()
	if err := checkProduct(&p); err != nil {
		return PharmacyProduct{}, err
	}
	if err := p.Validate(model.ActionCreate); err != nil {
		return PharmacyProduct{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		if err := checkSkuFree(tx, p.Sku); err != nil {
			return err
		}
		return tx.Create(&p)
	})
	if err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

func (m *Module) GetProduct(tenantId, id string) (PharmacyProduct, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	return getProduct(store, id)
}

func (m *Module) CreateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	if l.Id == "" {
		l.Id = m.ids.NewID()
	}
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionCreate); err != nil {
		return PharmacyLocation{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		if err := checkLocationNameFree(tx, l.Name); err != nil {
			return err
		}
		return tx.Create(&l)
	})
	if err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

func (m *Module) GetLocation(tenantId, id string) (PharmacyLocation, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	return getLocation(store, id)
}

// runAtomic ejecuta una operación de stock en una transacción del tenant dado y responde sus efectos sin publicarlos.
func (m *Module) runAtomic(tenantId string, op *PharmacyOperation, actor string) (*opEffects, error) {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return nil, err
	}
	var eff *opEffects
	err = store.Atomic(func(tx crud.Tx) error {
		var err error
		eff, err = m.runOperation(tx, op, actor)
		return err
	})
	return eff, err
}

func (m *Module) ReceiveStock(a ReceiveStockArgs) (PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationReceive, ProductId: a.ProductId, LocationId: a.LocationId, BatchNumber: a.BatchNumber,
		ExpirationDate: a.ExpirationDate, UnitCost: a.UnitCost, Quantity: a.Quantity,
		SourceDocumentType: a.SourceDocumentType, SourceDocumentId: a.SourceDocumentId, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return PharmacyMovement{}, err
	}
	if err := m.announce(eff); err != nil {
		return eff.movements[0], err
	}
	return eff.movements[0], nil
}

func (m *Module) RegisterEntry(a RegisterEntryArgs) (PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationEntry, ProductId: a.ProductId, LocationId: a.LocationId, BatchId: a.BatchId,
		Quantity: a.Quantity, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return PharmacyMovement{}, err
	}
	if err := m.announce(eff); err != nil {
		return eff.movements[0], err
	}
	return eff.movements[0], nil
}

// RegisterOutput: si el hecho de costo o la alerta no se pueden publicar después de guardar, los movimientos se devuelven
// CON el error: el cambio de stock ya está guardado (como siempre).
func (m *Module) RegisterOutput(a RegisterOutputArgs) ([]PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationOutput, ProductId: a.ProductId, LocationId: a.LocationId, BatchId: a.BatchId,
		Quantity: a.Quantity, MovementType: a.MovementType, SourceDocumentType: a.SourceDocumentType,
		SourceDocumentId: a.SourceDocumentId, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	return eff.movements, m.announce(eff)
}

func (m *Module) TransferStock(a TransferStockArgs) ([]PharmacyMovement, error) {
	op := PharmacyOperation{Kind: OperationTransfer, ProductId: a.ProductId, LocationId: a.FromLocationId,
		ToLocationId: a.ToLocationId, Quantity: a.Quantity, Note: a.Note}
	eff, err := m.runAtomic(a.TenantId, &op, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	return eff.movements, m.announce(eff)
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
