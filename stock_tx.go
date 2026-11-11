package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/notify"
	"webtyp.com/stock"
	"webtyp.com/storage"
	"webtyp.com/time"
)

// opEffects es lo que una operación de stock deja para anunciar cuando la escritura terminó.
type opEffects struct {
	movements []PharmacyMovement
	costs     []int64 // costo unitario del lote de cada movimiento, mismo orden que movements
	product   PharmacyProduct
	before    float64 // stock total del producto antes de una salida
	after     float64 // stock total del producto después de una salida
	isOutput  bool
}

func getProduct(tx crud.Tx, id string) (PharmacyProduct, error) {
	var p PharmacyProduct
	if err := tx.ReadOne(&p, id); err != nil {
		return PharmacyProduct{}, notFound(err, ErrProductNotFound)
	}
	return p, nil
}

func getLocation(tx crud.Tx, id string) (PharmacyLocation, error) {
	var l PharmacyLocation
	if err := tx.ReadOne(&l, id); err != nil {
		return PharmacyLocation{}, notFound(err, ErrLocationNotFound)
	}
	return l, nil
}

func getBatch(tx crud.Tx, id string) (PharmacyBatch, error) {
	var b PharmacyBatch
	if err := tx.ReadOne(&b, id); err != nil {
		return PharmacyBatch{}, notFound(err, ErrBatchNotFound)
	}
	return b, nil
}

// batchByNumber es el lote de un producto con ese número, y si existe.
func batchByNumber(tx crud.Tx, productId, number string) (PharmacyBatch, bool, error) {
	row, found, err := firstRow(tx, func() model.Model { return &PharmacyBatch{} },
		storage.Eq(PharmacyBatch_.ProductId, productId), storage.Eq(PharmacyBatch_.BatchNumber, number))
	if err != nil || !found {
		return PharmacyBatch{}, false, err
	}
	return *row.(*PharmacyBatch), true, nil
}

// levelAt es la fila de nivel de stock de un lote en una ubicación; found es falso cuando el lote nunca estuvo allí.
func levelAt(tx crud.Tx, locationId, batchId string) (PharmacyStockLevel, bool, error) {
	row, found, err := firstRow(tx, func() model.Model { return &PharmacyStockLevel{} },
		storage.Eq(PharmacyStockLevel_.LocationId, locationId), storage.Eq(PharmacyStockLevel_.BatchId, batchId))
	if err != nil || !found {
		return PharmacyStockLevel{}, false, err
	}
	return *row.(*PharmacyStockLevel), true, nil
}

// levelOrCreate devuelve la fila de nivel de stock de un lote en una ubicación, creándola en cantidad 0 y revision 0.
func (m *Module) levelOrCreate(tx crud.Tx, locationId, batchId string, now time.Nano) (PharmacyStockLevel, error) {
	level, found, err := levelAt(tx, locationId, batchId)
	if err != nil {
		return PharmacyStockLevel{}, err
	}
	if found {
		return level, nil
	}
	level = PharmacyStockLevel{Id: m.ids.NewID(), LocationId: locationId, BatchId: batchId, UpdatedAt: now}
	if err := tx.Create(&level); err != nil {
		return PharmacyStockLevel{}, err
	}
	return level, nil
}

// levelColumns son las columnas que cambia applyToLevel.
var levelColumns = []string{PharmacyStockLevel_.Quantity, PharmacyStockLevel_.Revision, PharmacyStockLevel_.UpdatedAt}

// applyToLevel cambia la cantidad de un nivel de stock por `quantity` en dirección d, guardado por la revisión que el
// llamador leyó: si la fila cambió desde entonces es ErrConflict y nada se escribe. stock.Apply mantiene la cantidad sin
// ser negativa (ErrInsufficientStock).
func applyToLevel(tx crud.Tx, level *PharmacyStockLevel, expectedRevision int64, d stock.Direction, quantity float64, now time.Nano) error {
	var current PharmacyStockLevel
	if err := tx.ReadOne(&current, level.Id); err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return ErrConflict
	}
	newQuantity, err := stock.Apply(current.Quantity, d, quantity)
	if err == stock.ErrInsufficient {
		return ErrInsufficientStock
	}
	if err != nil {
		return err
	}
	current.Quantity = newQuantity
	current.Revision++
	current.UpdatedAt = now
	if err := tx.UpdateFields(&current, current.Id, levelColumns); err != nil {
		return err
	}
	*level = current
	return nil
}

// addMovement añade una entrada al libro mayor dentro de la transacción.
func (m *Module) addMovement(tx crud.Tx, batchId, locationId string, t stock.Movement, quantity float64, docType, docId, note, by string, now time.Nano) (PharmacyMovement, error) {
	mv := PharmacyMovement{
		Id: m.ids.NewID(), BatchId: batchId, LocationId: locationId, MovementType: string(t), Quantity: quantity,
		SourceDocumentType: docType, SourceDocumentId: docId, Note: note, CreatedAt: now, CreatedBy: by,
	}
	if err := tx.Create(&mv); err != nil {
		return PharmacyMovement{}, err
	}
	return mv, nil
}

// lotsAt convierte el stock de un producto en una ubicación en los lotes que webtyp.com/stock asigna de: un Lot por lote
// que tiene una fila de nivel allí, con la caducidad del lote y la cantidad del nivel.
func lotsAt(tx crud.Tx, productId, locationId string) ([]stock.Lot, error) {
	rows, err := allRows(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, productId))
	if err != nil {
		return nil, err
	}
	lots := make([]stock.Lot, 0, len(rows))
	for _, r := range rows {
		b := r.(*PharmacyBatch)
		level, found, err := levelAt(tx, locationId, b.Id)
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

// totalOf es el stock del producto sumado en cada ubicación y lote.
func totalOf(tx crud.Tx, productId string) (float64, error) {
	batches, err := allRows(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, productId))
	if err != nil {
		return 0, err
	}
	var total float64
	for _, b := range batches {
		levels, err := allRows(tx, func() model.Model { return &PharmacyStockLevel{} }, storage.Eq(PharmacyStockLevel_.BatchId, b.(*PharmacyBatch).Id))
		if err != nil {
			return 0, err
		}
		for _, l := range levels {
			if q := l.(*PharmacyStockLevel).Quantity; q > 0 {
				total += q
			}
		}
	}
	return total, nil
}

// runOperation ejecuta una operación de stock dentro de tx: según op.Kind llama a la lógica de receive, entry, output o
// transfer. La comparten la regla Create de pharmacy_operation y los cuatro métodos que se conservan (ReceiveStock…).
// actor es quien la hace (movement.created_by). No publica nada: lo hace announce cuando la escritura terminó.
//
// Ejemplo: eff, err := m.runOperation(tx, &PharmacyOperation{Kind: OperationReceive, …}, "u1")
func (m *Module) runOperation(tx crud.Tx, op *PharmacyOperation, actor string) (*opEffects, error) {
	now := time.NowNano()
	switch op.Kind {
	case OperationReceive:
		return m.receive(tx, op, actor, now)
	case OperationEntry:
		return m.entry(tx, op, actor, now)
	case OperationOutput:
		return m.output(tx, op, actor, now)
	case OperationTransfer:
		return m.transfer(tx, op, actor, now)
	}
	return nil, &model.FieldError{Field: PharmacyOperation_.Kind, Err: ErrOperationKind}
}

// receive añade stock de un producto a una ubicación: encuentra o crea el lote (la fecha de caducidad es obligatoria),
// añade la cantidad al nivel (location, batch) y añade un movimiento stock.Purchase.
func (m *Module) receive(tx crud.Tx, op *PharmacyOperation, actor string, now time.Nano) (*opEffects, error) {
	if op.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	if op.BatchNumber == "" {
		return nil, ErrBatchNumber
	}
	if op.ExpirationDate <= 0 {
		return nil, ErrExpiryRequired
	}
	if op.UnitCost < 0 {
		return nil, ErrUnitCost
	}
	product, err := getProduct(tx, op.ProductId)
	if err != nil {
		return nil, err
	}
	if !product.IsActive {
		return nil, ErrProductInactive
	}
	location, err := getLocation(tx, op.LocationId)
	if err != nil {
		return nil, err
	}
	if !location.IsActive {
		return nil, ErrLocationInactive
	}
	batch, found, err := batchByNumber(tx, op.ProductId, op.BatchNumber)
	if err != nil {
		return nil, err
	}
	if found {
		if batch.ExpirationDate != op.ExpirationDate {
			return nil, ErrBatchExpiryMismatch
		}
		if op.UnitCost != 0 && batch.UnitCost != 0 && batch.UnitCost != op.UnitCost {
			return nil, ErrBatchCostMismatch
		}
	} else {
		batch = PharmacyBatch{
			Id: m.ids.NewID(), ProductId: op.ProductId, BatchNumber: op.BatchNumber,
			ExpirationDate: op.ExpirationDate, UnitCost: op.UnitCost, IsActive: true, UpdatedAt: now,
		}
		if err := tx.Create(&batch); err != nil {
			return nil, err
		}
	}
	level, err := m.levelOrCreate(tx, op.LocationId, batch.Id, now)
	if err != nil {
		return nil, err
	}
	if err := applyToLevel(tx, &level, level.Revision, stock.In, op.Quantity, now); err != nil {
		return nil, err
	}
	mv, err := m.addMovement(tx, batch.Id, op.LocationId, stock.Purchase, op.Quantity, op.SourceDocumentType, op.SourceDocumentId, op.Note, actor, now)
	if err != nil {
		return nil, err
	}
	return &opEffects{movements: []PharmacyMovement{mv}}, nil
}

// entry añade a un lote EXISTENTE: una corrección entrante de un conteo (stock.AdjustIn).
func (m *Module) entry(tx crud.Tx, op *PharmacyOperation, actor string, now time.Nano) (*opEffects, error) {
	if op.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	if _, err := getProduct(tx, op.ProductId); err != nil {
		return nil, err
	}
	if _, err := getLocation(tx, op.LocationId); err != nil {
		return nil, err
	}
	batch, err := getBatch(tx, op.BatchId)
	if err != nil {
		return nil, err
	}
	if batch.ProductId != op.ProductId {
		return nil, ErrBatchNotFound
	}
	level, err := m.levelOrCreate(tx, op.LocationId, batch.Id, now)
	if err != nil {
		return nil, err
	}
	if err := applyToLevel(tx, &level, level.Revision, stock.In, op.Quantity, now); err != nil {
		return nil, err
	}
	mv, err := m.addMovement(tx, batch.Id, op.LocationId, stock.AdjustIn, op.Quantity, "", "", op.Note, actor, now)
	if err != nil {
		return nil, err
	}
	return &opEffects{movements: []PharmacyMovement{mv}}, nil
}

// output saca stock: una venta, uso interno o descarte. Sin un id de lote, los lotes se toman por primero en caducar
// primero en salir (stock.Allocate); solo un descarte (stock.AdjustOut) puede tomar stock caducado. Produce un movimiento
// por lote usado.
func (m *Module) output(tx crud.Tx, op *PharmacyOperation, actor string, now time.Nano) (*opEffects, error) {
	if op.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	t := stock.Movement(op.MovementType)
	if t != stock.Sale && t != stock.InternalUse && t != stock.AdjustOut {
		return nil, ErrMovementType
	}
	before, err := totalOf(tx, op.ProductId)
	if err != nil {
		return nil, err
	}
	product, err := getProduct(tx, op.ProductId)
	if err != nil {
		return nil, err
	}
	if t == stock.Sale && !product.ForSale {
		return nil, ErrNotForSale
	}
	if _, err := getLocation(tx, op.LocationId); err != nil {
		return nil, err
	}

	var takes []stock.Take
	if op.BatchId == "" {
		lots, err := lotsAt(tx, op.ProductId, op.LocationId)
		if err != nil {
			return nil, err
		}
		var allocateAt time.Unix
		if t == stock.AdjustOut {
			allocateAt = 0 // a write-off may take expired stock: with now = 0 no lot counts as expired
		} else {
			allocateAt = time.NowUnix()
		}
		takes, err = stock.Allocate(lots, op.Quantity, allocateAt)
		if err == stock.ErrInsufficient {
			return nil, ErrInsufficientStock
		}
		if err != nil {
			return nil, err
		}
	} else {
		batch, err := getBatch(tx, op.BatchId)
		if err != nil {
			return nil, err
		}
		if batch.ProductId != op.ProductId {
			return nil, ErrBatchNotFound
		}
		level, found, err := levelAt(tx, op.LocationId, batch.Id)
		if err != nil {
			return nil, err
		}
		if !found || level.Quantity < op.Quantity {
			return nil, ErrInsufficientStock
		}
		if t != stock.AdjustOut && (stock.Lot{Expiry: batch.ExpirationDate}).Expired(time.NowUnix()) {
			return nil, ErrBatchExpired
		}
		takes = []stock.Take{{LotID: batch.Id, Quantity: op.Quantity}}
	}

	eff := &opEffects{product: product, before: before, isOutput: true}
	for _, take := range takes {
		batch, err := getBatch(tx, take.LotID)
		if err != nil {
			return nil, err
		}
		level, found, err := levelAt(tx, op.LocationId, take.LotID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrInsufficientStock
		}
		if err := applyToLevel(tx, &level, level.Revision, stock.Out, take.Quantity, now); err != nil {
			return nil, err
		}
		mv, err := m.addMovement(tx, take.LotID, op.LocationId, t, take.Quantity, op.SourceDocumentType, op.SourceDocumentId, op.Note, actor, now)
		if err != nil {
			return nil, err
		}
		eff.movements = append(eff.movements, mv)
		eff.costs = append(eff.costs, batch.UnitCost)
	}
	after, err := totalOf(tx, op.ProductId)
	if err != nil {
		return nil, err
	}
	eff.after = after
	return eff, nil
}

// transfer mueve stock de un producto entre dos ubicaciones, extrayendo los lotes por primero en caducar primero en salir
// y poniendo los mismos lotes en el destino: movimientos stock.TransferOut en el origen, stock.TransferIn en el destino.
func (m *Module) transfer(tx crud.Tx, op *PharmacyOperation, actor string, now time.Nano) (*opEffects, error) {
	if op.Quantity <= 0 {
		return nil, &model.FieldError{Field: "quantity", Err: stock.ErrQuantity}
	}
	if op.LocationId == op.ToLocationId {
		return nil, ErrSameLocation
	}
	if _, err := getProduct(tx, op.ProductId); err != nil {
		return nil, err
	}
	if _, err := getLocation(tx, op.LocationId); err != nil {
		return nil, err
	}
	to, err := getLocation(tx, op.ToLocationId)
	if err != nil {
		return nil, err
	}
	if !to.IsActive {
		return nil, ErrLocationInactive
	}
	lots, err := lotsAt(tx, op.ProductId, op.LocationId)
	if err != nil {
		return nil, err
	}
	takes, err := stock.Allocate(lots, op.Quantity, time.NowUnix())
	if err == stock.ErrInsufficient {
		return nil, ErrInsufficientStock
	}
	if err != nil {
		return nil, err
	}
	eff := &opEffects{}
	for _, take := range takes {
		src, found, err := levelAt(tx, op.LocationId, take.LotID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrInsufficientStock
		}
		if err := applyToLevel(tx, &src, src.Revision, stock.Out, take.Quantity, now); err != nil {
			return nil, err
		}
		dst, err := m.levelOrCreate(tx, op.ToLocationId, take.LotID, now)
		if err != nil {
			return nil, err
		}
		if err := applyToLevel(tx, &dst, dst.Revision, stock.In, take.Quantity, now); err != nil {
			return nil, err
		}
		mvOut, err := m.addMovement(tx, take.LotID, op.LocationId, stock.TransferOut, take.Quantity, "", "", op.Note, actor, now)
		if err != nil {
			return nil, err
		}
		mvIn, err := m.addMovement(tx, take.LotID, op.ToLocationId, stock.TransferIn, take.Quantity, "", "", op.Note, actor, now)
		if err != nil {
			return nil, err
		}
		eff.movements = append(eff.movements, mvOut, mvIn)
	}
	return eff, nil
}

// movementIds une los ids de los movimientos con comas (pharmacy_operation.movement_ids).
func movementIds(movements []PharmacyMovement) string {
	ids := ""
	for i, mv := range movements {
		if i > 0 {
			ids += ","
		}
		ids += mv.Id
	}
	return ids
}

// announce publica lo que una operación terminada dejó: un evento por movimiento, los hechos de costo de una salida y la
// alerta de mínimo cruzado. Si el hecho de costo o la alerta fallan, el cambio de stock YA está guardado: el error se
// devuelve para que quien llama lo vea (los movimientos siguen siendo válidos).
// Ejemplo: err := m.announce(eff)
func (m *Module) announce(eff *opEffects) error {
	for _, mv := range eff.movements {
		m.publish(mv)
	}
	if !eff.isOutput {
		return nil
	}
	if err := m.publishCosts(eff.movements, eff.costs); err != nil {
		return err
	}
	if len(eff.movements) == 0 {
		return nil
	}
	if stock.CrossedLow(eff.before, eff.after, eff.product.MinQuantity) {
		return m.raise(notify.Alert{
			Code:    AlertBelowMinimum,
			Subject: eff.product.Name,
			Body:    fmt.Sprintf(msgBelowMinimum, eff.after, eff.product.Unit, eff.product.MinQuantity),
			Source:  notify.Ref{Type: alertSourceProduct, ID: eff.product.Id},
		})
	}
	return nil
}
