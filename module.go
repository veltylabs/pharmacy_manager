package pharmacymanager

import (
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/money"
	"webtyp.com/notify"
	"webtyp.com/orm"
	"webtyp.com/stock"
	"webtyp.com/time"
)

// TopicMovementRegistered is published once per stock movement, after the transaction commits. Payload: *PharmacyMovement.
const TopicMovementRegistered = "pharmacy.movement.registered"

const (
	// sourceType is the Ref.Type of the cost facts: the movement is their cause.
	sourceType = "pharmacy_movement"

	msgNeedIDs      = "pharmacy: Deps.IDs is required"
	msgPublishCost  = "pharmacy: the stock change was saved but its cost fact could not be published:"
	msgBelowMinimum = "%v %s left; minimum %v."
	msgExpiring     = "%d expired batches and %d expiring within %d days."
)

// Deps are what the module needs from the application that composes it.
type Deps struct {
	IDs        model.IDGenerator // required: the module never builds a generator
	Publisher  events.Publisher  // optional: nil disables the events and the cost facts
	MoneyScope string            // optional: the area of the business the cost facts belong to; "" disables the cost facts
	AlertArea  string            // optional: the application's notify.Area code for this module's alerts; "" disables them
}

type Module struct {
	db        *orm.DB
	ids       model.IDGenerator
	pub       events.Publisher
	scope     string
	alertArea string
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err(msgNeedIDs)
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher, scope: deps.MoneyScope, alertArea: deps.AlertArea}, nil
}

func (m *Module) ModelName() string { return ModelName }

// ---- products and locations (the pattern the rest of the CRUD follows) ----

func (m *Module) CreateProduct(p PharmacyProduct) (PharmacyProduct, error) {
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
	if _, found, err := m.productBySKU(m.db, p.TenantId, p.Sku); err != nil {
		return PharmacyProduct{}, err
	} else if found {
		return PharmacyProduct{}, ErrSkuExists
	}
	if err := m.db.Create(&p); err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

// checkProduct applies the rules of a product that the field constraints cannot express.
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

func (m *Module) GetProduct(tenantId, id string) (PharmacyProduct, error) {
	return getProduct(m.db, tenantId, id)
}

func getProduct(db *orm.DB, tenantId, id string) (PharmacyProduct, error) {
	var p PharmacyProduct
	qb := db.Query(&p).Where(PharmacyProduct_.Id).Eq(id).Where(PharmacyProduct_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyProduct(qb, &p)
	if orm.IsNotFound(err) {
		return PharmacyProduct{}, ErrProductNotFound
	}
	if err != nil {
		return PharmacyProduct{}, err
	}
	return *got, nil
}

func (m *Module) productBySKU(db *orm.DB, tenantId, sku string) (PharmacyProduct, bool, error) {
	var p PharmacyProduct
	qb := db.Query(&p).Where(PharmacyProduct_.Sku).Eq(sku).Where(PharmacyProduct_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyProduct(qb, &p)
	if orm.IsNotFound(err) {
		return PharmacyProduct{}, false, nil
	}
	if err != nil {
		return PharmacyProduct{}, false, err
	}
	return *got, true, nil
}

func (m *Module) CreateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	if l.Id == "" {
		l.Id = m.ids.NewID()
	}
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionCreate); err != nil {
		return PharmacyLocation{}, err
	}
	var proxy PharmacyLocation
	qb := m.db.Query(&proxy).Where(PharmacyLocation_.Name).Eq(l.Name).Where(PharmacyLocation_.TenantId).Eq(l.TenantId)
	if _, err := ReadOnePharmacyLocation(qb, &proxy); err == nil {
		return PharmacyLocation{}, ErrLocationNameExists
	} else if !orm.IsNotFound(err) {
		return PharmacyLocation{}, err
	}
	if err := m.db.Create(&l); err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

func (m *Module) GetLocation(tenantId, id string) (PharmacyLocation, error) {
	return getLocation(m.db, tenantId, id)
}

func getLocation(db *orm.DB, tenantId, id string) (PharmacyLocation, error) {
	var l PharmacyLocation
	qb := db.Query(&l).Where(PharmacyLocation_.Id).Eq(id).Where(PharmacyLocation_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyLocation(qb, &l)
	if orm.IsNotFound(err) {
		return PharmacyLocation{}, ErrLocationNotFound
	}
	if err != nil {
		return PharmacyLocation{}, err
	}
	return *got, nil
}

// ---- batches and stock levels ----

func getBatch(db *orm.DB, tenantId, id string) (PharmacyBatch, error) {
	var b PharmacyBatch
	qb := db.Query(&b).Where(PharmacyBatch_.Id).Eq(id).Where(PharmacyBatch_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyBatch(qb, &b)
	if orm.IsNotFound(err) {
		return PharmacyBatch{}, ErrBatchNotFound
	}
	if err != nil {
		return PharmacyBatch{}, err
	}
	return *got, nil
}

func batchByNumber(db *orm.DB, tenantId, productId, number string) (PharmacyBatch, bool, error) {
	var b PharmacyBatch
	qb := db.Query(&b).Where(PharmacyBatch_.ProductId).Eq(productId).Where(PharmacyBatch_.BatchNumber).Eq(number).Where(PharmacyBatch_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyBatch(qb, &b)
	if orm.IsNotFound(err) {
		return PharmacyBatch{}, false, nil
	}
	if err != nil {
		return PharmacyBatch{}, false, err
	}
	return *got, true, nil
}

// levelAt is the stock level row of a batch in a location; found is false when the batch was never there.
func levelAt(db *orm.DB, tenantId, locationId, batchId string) (PharmacyStockLevel, bool, error) {
	var l PharmacyStockLevel
	qb := db.Query(&l).Where(PharmacyStockLevel_.LocationId).Eq(locationId).Where(PharmacyStockLevel_.BatchId).Eq(batchId).Where(PharmacyStockLevel_.TenantId).Eq(tenantId)
	got, err := ReadOnePharmacyStockLevel(qb, &l)
	if orm.IsNotFound(err) {
		return PharmacyStockLevel{}, false, nil
	}
	if err != nil {
		return PharmacyStockLevel{}, false, err
	}
	return *got, true, nil
}

// levelOrCreate returns the level row of a batch in a location, creating it at quantity 0 and revision 0.
func (m *Module) levelOrCreate(tx *orm.DB, tenantId, locationId, batchId string, now time.Nano) (PharmacyStockLevel, error) {
	level, found, err := levelAt(tx, tenantId, locationId, batchId)
	if err != nil {
		return PharmacyStockLevel{}, err
	}
	if found {
		return level, nil
	}
	level = PharmacyStockLevel{Id: m.ids.NewID(), TenantId: tenantId, LocationId: locationId, BatchId: batchId, UpdatedAt: now}
	if err := tx.Create(&level); err != nil {
		return PharmacyStockLevel{}, err
	}
	return level, nil
}

// applyToLevel changes the quantity of a stock level by `quantity` in direction d, guarded by the revision the caller
// read: if the row changed since (expectedRevision differs from the stored one) it is ErrConflict and nothing is
// written. stock.Apply keeps the quantity from going negative (ErrInsufficientStock).
func applyToLevel(tx *orm.DB, level *PharmacyStockLevel, expectedRevision int64, d stock.Direction, quantity float64, now time.Nano) error {
	var current PharmacyStockLevel
	qb := tx.Query(&current).Where(PharmacyStockLevel_.Id).Eq(level.Id).Where(PharmacyStockLevel_.TenantId).Eq(level.TenantId)
	got, err := ReadOnePharmacyStockLevel(qb, &current)
	if err != nil {
		return err
	}
	if got.Revision != expectedRevision {
		return ErrConflict
	}
	newQuantity, err := stock.Apply(got.Quantity, d, quantity)
	if err == stock.ErrInsufficient {
		return ErrInsufficientStock
	}
	if err != nil {
		return err
	}
	got.Quantity = newQuantity
	got.Revision++
	got.UpdatedAt = now
	if err := tx.Update(got, orm.Eq(PharmacyStockLevel_.Id, got.Id), orm.Eq(PharmacyStockLevel_.TenantId, got.TenantId)); err != nil {
		return err
	}
	*level = *got
	return nil
}

// addMovement appends one ledger entry inside the transaction.
func (m *Module) addMovement(tx *orm.DB, tenantId, batchId, locationId string, t stock.Movement, quantity float64, docType, docId, note, by string, now time.Nano) (PharmacyMovement, error) {
	mv := PharmacyMovement{
		Id: m.ids.NewID(), TenantId: tenantId, BatchId: batchId, LocationId: locationId, MovementType: string(t), Quantity: quantity,
		SourceDocumentType: docType, SourceDocumentId: docId, Note: note, CreatedAt: now, CreatedBy: by,
	}
	if err := tx.Create(&mv); err != nil {
		return PharmacyMovement{}, err
	}
	return mv, nil
}

// lotsAt turns the stock of a product in a location into the lots webtyp.com/stock allocates from: one Lot per batch
// that has a level row there, with the batch's expiry and the level's quantity.
func lotsAt(db *orm.DB, tenantId, productId, locationId string) ([]stock.Lot, error) {
	var proxy PharmacyBatch
	batches, err := ReadAllPharmacyBatch(db.Query(&proxy).Where(PharmacyBatch_.ProductId).Eq(productId).Where(PharmacyBatch_.TenantId).Eq(tenantId))
	if err != nil {
		return nil, err
	}
	lots := make([]stock.Lot, 0, len(batches))
	for _, b := range batches {
		level, found, err := levelAt(db, tenantId, locationId, b.Id)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		lots = append(lots, stock.Lot{ID: b.Id, Expiry: b.ExpirationDate, Quantity: level.Quantity})
	}
	return lots, nil
}

// ---- the four stock operations ----

// ReceiveStock adds stock of a product to a location: it finds or creates the batch (the expiry date is mandatory), adds
// the quantity to the (location, batch) level and appends a stock.Purchase movement, all in one transaction.
func (m *Module) ReceiveStock(a ReceiveStockArgs) (PharmacyMovement, error) {
	if a.Quantity <= 0 {
		return PharmacyMovement{}, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	if a.BatchNumber == "" {
		return PharmacyMovement{}, ErrBatchNumber
	}
	if a.ExpirationDate <= 0 {
		return PharmacyMovement{}, ErrExpiryRequired
	}
	if a.UnitCost < 0 {
		return PharmacyMovement{}, ErrUnitCost
	}
	now := time.NowNano()
	var mv PharmacyMovement
	err := m.db.Tx(func(tx *orm.DB) error {
		product, err := getProduct(tx, a.TenantId, a.ProductId)
		if err != nil {
			return err
		}
		if !product.IsActive {
			return ErrProductInactive
		}
		location, err := getLocation(tx, a.TenantId, a.LocationId)
		if err != nil {
			return err
		}
		if !location.IsActive {
			return ErrLocationInactive
		}
		batch, found, err := batchByNumber(tx, a.TenantId, a.ProductId, a.BatchNumber)
		if err != nil {
			return err
		}
		if found {
			if batch.ExpirationDate != a.ExpirationDate {
				return ErrBatchExpiryMismatch
			}
			if a.UnitCost != 0 && batch.UnitCost != 0 && batch.UnitCost != a.UnitCost {
				return ErrBatchCostMismatch
			}
		} else {
			batch = PharmacyBatch{
				Id: m.ids.NewID(), TenantId: a.TenantId, ProductId: a.ProductId, BatchNumber: a.BatchNumber,
				ExpirationDate: a.ExpirationDate, UnitCost: a.UnitCost, IsActive: true, UpdatedAt: now,
			}
			if err := tx.Create(&batch); err != nil {
				return err
			}
		}
		level, err := m.levelOrCreate(tx, a.TenantId, a.LocationId, batch.Id, now)
		if err != nil {
			return err
		}
		if err := applyToLevel(tx, &level, level.Revision, stock.In, a.Quantity, now); err != nil {
			return err
		}
		mv, err = m.addMovement(tx, a.TenantId, batch.Id, a.LocationId, stock.Purchase, a.Quantity, a.SourceDocumentType, a.SourceDocumentId, a.Note, a.CreatedBy, now)
		return err
	})
	if err != nil {
		return PharmacyMovement{}, err
	}
	m.publish(mv)
	return mv, nil
}

// RegisterEntry adds to an EXISTING batch: an inbound correction of a count (stock.AdjustIn).
func (m *Module) RegisterEntry(a RegisterEntryArgs) (PharmacyMovement, error) {
	if a.Quantity <= 0 {
		return PharmacyMovement{}, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	now := time.NowNano()
	var mv PharmacyMovement
	err := m.db.Tx(func(tx *orm.DB) error {
		if _, err := getProduct(tx, a.TenantId, a.ProductId); err != nil {
			return err
		}
		if _, err := getLocation(tx, a.TenantId, a.LocationId); err != nil {
			return err
		}
		batch, err := getBatch(tx, a.TenantId, a.BatchId)
		if err != nil {
			return err
		}
		if batch.ProductId != a.ProductId {
			return ErrBatchNotFound
		}
		level, err := m.levelOrCreate(tx, a.TenantId, a.LocationId, batch.Id, now)
		if err != nil {
			return err
		}
		if err := applyToLevel(tx, &level, level.Revision, stock.In, a.Quantity, now); err != nil {
			return err
		}
		mv, err = m.addMovement(tx, a.TenantId, batch.Id, a.LocationId, stock.AdjustIn, a.Quantity, "", "", a.Note, a.CreatedBy, now)
		return err
	})
	if err != nil {
		return PharmacyMovement{}, err
	}
	m.publish(mv)
	return mv, nil
}

// RegisterOutput takes stock out: a sale, an internal use or a write-off. Without a batch id the batches are drawn by
// first expiry first out (stock.Allocate); only a write-off (stock.AdjustOut) may take expired stock. It returns one
// movement per batch used. If the cost fact cannot be published after the commit, the movements are returned WITH the
// error: the stock change is already saved.
func (m *Module) RegisterOutput(a RegisterOutputArgs) ([]PharmacyMovement, error) {
	if a.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	t := stock.Movement(a.MovementType)
	if t != stock.Sale && t != stock.InternalUse && t != stock.AdjustOut {
		return nil, ErrMovementType
	}
	before, err := m.totalProduct(a.TenantId, a.ProductId)
	if err != nil {
		return nil, err
	}
	now := time.NowNano()
	var out []PharmacyMovement
	var costs []int64 // unit cost of the batch of each movement, same order as out
	err = m.db.Tx(func(tx *orm.DB) error {
		product, err := getProduct(tx, a.TenantId, a.ProductId)
		if err != nil {
			return err
		}
		if t == stock.Sale && !product.ForSale {
			return ErrNotForSale
		}
		if _, err := getLocation(tx, a.TenantId, a.LocationId); err != nil {
			return err
		}

		var takes []stock.Take
		if a.BatchId == "" {
			lots, err := lotsAt(tx, a.TenantId, a.ProductId, a.LocationId)
			if err != nil {
				return err
			}
			var allocateAt time.Unix
			if t == stock.AdjustOut {
				allocateAt = 0 // a write-off may take expired stock: with now = 0 no lot counts as expired
			} else {
				allocateAt = time.NowUnix()
			}
			takes, err = stock.Allocate(lots, a.Quantity, allocateAt)
			if err == stock.ErrInsufficient {
				return ErrInsufficientStock
			}
			if err != nil {
				return err
			}
		} else {
			batch, err := getBatch(tx, a.TenantId, a.BatchId)
			if err != nil {
				return err
			}
			if batch.ProductId != a.ProductId {
				return ErrBatchNotFound
			}
			level, found, err := levelAt(tx, a.TenantId, a.LocationId, batch.Id)
			if err != nil {
				return err
			}
			if !found || level.Quantity < a.Quantity {
				return ErrInsufficientStock
			}
			if t != stock.AdjustOut && (stock.Lot{Expiry: batch.ExpirationDate}).Expired(time.NowUnix()) {
				return ErrBatchExpired
			}
			takes = []stock.Take{{LotID: batch.Id, Quantity: a.Quantity}}
		}

		for _, take := range takes {
			batch, err := getBatch(tx, a.TenantId, take.LotID)
			if err != nil {
				return err
			}
			level, found, err := levelAt(tx, a.TenantId, a.LocationId, take.LotID)
			if err != nil {
				return err
			}
			if !found {
				return ErrInsufficientStock
			}
			if err := applyToLevel(tx, &level, level.Revision, stock.Out, take.Quantity, now); err != nil {
				return err
			}
			mv, err := m.addMovement(tx, a.TenantId, take.LotID, a.LocationId, t, take.Quantity, a.SourceDocumentType, a.SourceDocumentId, a.Note, a.CreatedBy, now)
			if err != nil {
				return err
			}
			out = append(out, mv)
			costs = append(costs, batch.UnitCost)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, mv := range out {
		m.publish(mv)
	}
	if err := m.publishCosts(out, costs); err != nil {
		return out, err
	}
	if len(out) > 0 {
		product, err := m.GetProduct(a.TenantId, a.ProductId)
		if err != nil {
			return out, err
		}
		after, err := m.totalProduct(a.TenantId, a.ProductId)
		if err != nil {
			return out, err
		}
		if stock.CrossedLow(before, after, product.MinQuantity) {
			alertErr := m.raise(notify.Alert{
				Code:    AlertBelowMinimum,
				Subject: product.Name,
				Body:    fmt.Sprintf(msgBelowMinimum, after, product.Unit, product.MinQuantity),
				Source:  notify.Ref{Type: alertSourceProduct, ID: product.Id},
			})
			if alertErr != nil {
				return out, alertErr
			}
		}
	}
	return out, nil
}

// TransferStock moves stock of a product between two locations, drawing the batches by first expiry first out and
// putting the same batches in the destination: stock.TransferOut movements at the origin, stock.TransferIn at the
// destination, one transaction.
func (m *Module) TransferStock(a TransferStockArgs) ([]PharmacyMovement, error) {
	if a.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	if a.FromLocationId == a.ToLocationId {
		return nil, ErrSameLocation
	}
	now := time.NowNano()
	var out []PharmacyMovement
	err := m.db.Tx(func(tx *orm.DB) error {
		if _, err := getProduct(tx, a.TenantId, a.ProductId); err != nil {
			return err
		}
		if _, err := getLocation(tx, a.TenantId, a.FromLocationId); err != nil {
			return err
		}
		to, err := getLocation(tx, a.TenantId, a.ToLocationId)
		if err != nil {
			return err
		}
		if !to.IsActive {
			return ErrLocationInactive
		}
		lots, err := lotsAt(tx, a.TenantId, a.ProductId, a.FromLocationId)
		if err != nil {
			return err
		}
		takes, err := stock.Allocate(lots, a.Quantity, time.NowUnix())
		if err == stock.ErrInsufficient {
			return ErrInsufficientStock
		}
		if err != nil {
			return err
		}
		for _, take := range takes {
			src, found, err := levelAt(tx, a.TenantId, a.FromLocationId, take.LotID)
			if err != nil {
				return err
			}
			if !found {
				return ErrInsufficientStock
			}
			if err := applyToLevel(tx, &src, src.Revision, stock.Out, take.Quantity, now); err != nil {
				return err
			}
			dst, err := m.levelOrCreate(tx, a.TenantId, a.ToLocationId, take.LotID, now)
			if err != nil {
				return err
			}
			if err := applyToLevel(tx, &dst, dst.Revision, stock.In, take.Quantity, now); err != nil {
				return err
			}
			mvOut, err := m.addMovement(tx, a.TenantId, take.LotID, a.FromLocationId, stock.TransferOut, take.Quantity, "", "", a.Note, a.CreatedBy, now)
			if err != nil {
				return err
			}
			mvIn, err := m.addMovement(tx, a.TenantId, take.LotID, a.ToLocationId, stock.TransferIn, take.Quantity, "", "", a.Note, a.CreatedBy, now)
			if err != nil {
				return err
			}
			out = append(out, mvOut, mvIn)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, mv := range out {
		m.publish(mv)
	}
	return out, nil
}

// ---- events and cost facts ----

func (m *Module) publish(mv PharmacyMovement) {
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicMovementRegistered, Payload: &mv})
	}
}

// publishCosts announces what an output consumed, at cost: one money.Consumed fact per movement whose batch has a unit cost.
// It does nothing unless the application gave a Publisher and a MoneyScope.
func (m *Module) publishCosts(movements []PharmacyMovement, unitCosts []int64) error {
	if m.pub == nil || m.scope == "" {
		return nil
	}
	for i, mv := range movements {
		amount := int64(float64(unitCosts[i])*mv.Quantity + 0.5) // half away from zero; both factors are positive
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
