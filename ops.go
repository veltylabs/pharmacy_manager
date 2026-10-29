package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/router"
)

// Operation names, as the transport and the view see them: "<ModelName>.<name>".
const (
	OpListProducts      = "list_products"
	OpGetProduct        = "get_product"
	OpCreateProduct     = "create_product"
	OpUpdateProduct     = "update_product"
	OpUpsertProduct     = "upsert_product"
	OpDeactivateProduct = "deactivate_product"
	OpDeleteProduct     = "delete_product"

	OpListLocations  = "list_locations"
	OpUpsertLocation = "upsert_location"
	OpUpdateLocation = "update_location"
	OpDeleteLocation = "delete_location"

	OpListBatches    = "list_batches"
	OpReceiveStock   = "receive_stock"
	OpRegisterEntry  = "register_entry"
	OpRegisterOutput = "register_output"
	OpTransferStock  = "transfer_stock"
	OpListStock      = "list_stock"
	OpListMovements  = "list_movements"
	OpListExpiring   = "list_expiring"
	OpListLowStock   = "list_low_stock"
)

// Resources of the RBAC: one per table, so that a role can be granted exactly one.
const (
	resProduct  = "pharmacy_product"
	resLocation = "pharmacy_location"
	resBatch    = "pharmacy_batch"
	resLevel    = "pharmacy_stock_level"
	resMovement = "pharmacy_movement"
)

// ---- the wire shapes of the operations the generic view drives (webtyp.com/view sends exactly these) ----

// UpsertProductArgs is what view's save sends: {"records": [...]}.
type UpsertProductArgs struct{ Records []PharmacyProduct }

func (a *UpsertProductArgs) ModelName() string     { return "upsert_product_args" }
func (a *UpsertProductArgs) Schema() []model.Field { return nil }
func (a *UpsertProductArgs) Pointers() []any       { return nil }
func (a *UpsertProductArgs) IsNil() bool           { return a == nil }
func (a *UpsertProductArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}
func (a *UpsertProductArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("records"); ok {
		a.Records = make([]PharmacyProduct, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &a.Records[i])
		}
	}
}

// UpsertLocationArgs is what view's save sends for a location.
type UpsertLocationArgs struct{ Records []PharmacyLocation }

func (a *UpsertLocationArgs) ModelName() string     { return "upsert_location_args" }
func (a *UpsertLocationArgs) Schema() []model.Field { return nil }
func (a *UpsertLocationArgs) Pointers() []any       { return nil }
func (a *UpsertLocationArgs) IsNil() bool           { return a == nil }
func (a *UpsertLocationArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
}
func (a *UpsertLocationArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("records"); ok {
		a.Records = make([]PharmacyLocation, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &a.Records[i])
		}
	}
}

// UpdateProductArgs is what view's update sends: the ids, the names of the fields to change and a record with their values.
type UpdateProductArgs struct {
	Ids    []string
	Fields []string
	Record PharmacyProduct
}

func (a *UpdateProductArgs) ModelName() string     { return "update_product_args" }
func (a *UpdateProductArgs) Schema() []model.Field { return nil }
func (a *UpdateProductArgs) Pointers() []any       { return nil }
func (a *UpdateProductArgs) IsNil() bool           { return a == nil }
func (a *UpdateProductArgs) EncodeFields(w model.FieldWriter) {
	writeStrings(w, "ids", a.Ids)
	writeStrings(w, "fields", a.Fields)
	w.Object("record", &a.Record)
}
func (a *UpdateProductArgs) DecodeFields(r model.FieldReader) {
	a.Ids = readStrings(r, "ids")
	a.Fields = readStrings(r, "fields")
	r.Object("record", &a.Record)
}

// UpdateLocationArgs is the same shape for a location.
type UpdateLocationArgs struct {
	Ids    []string
	Fields []string
	Record PharmacyLocation
}

func (a *UpdateLocationArgs) ModelName() string     { return "update_location_args" }
func (a *UpdateLocationArgs) Schema() []model.Field { return nil }
func (a *UpdateLocationArgs) Pointers() []any       { return nil }
func (a *UpdateLocationArgs) IsNil() bool           { return a == nil }
func (a *UpdateLocationArgs) EncodeFields(w model.FieldWriter) {
	writeStrings(w, "ids", a.Ids)
	writeStrings(w, "fields", a.Fields)
	w.Object("record", &a.Record)
}
func (a *UpdateLocationArgs) DecodeFields(r model.FieldReader) {
	a.Ids = readStrings(r, "ids")
	a.Fields = readStrings(r, "fields")
	r.Object("record", &a.Record)
}

// DeleteIdsArgs is what view's delete sends for products and locations: {"ids": [...]}.
type DeleteIdsArgs struct{ Ids []string }

func (a *DeleteIdsArgs) ModelName() string                { return "delete_ids_args" }
func (a *DeleteIdsArgs) Schema() []model.Field            { return nil }
func (a *DeleteIdsArgs) Pointers() []any                  { return nil }
func (a *DeleteIdsArgs) IsNil() bool                      { return a == nil }
func (a *DeleteIdsArgs) EncodeFields(w model.FieldWriter) { writeStrings(w, "ids", a.Ids) }
func (a *DeleteIdsArgs) DecodeFields(r model.FieldReader) { a.Ids = readStrings(r, "ids") }

func writeStrings(w model.FieldWriter, name string, values []string) {
	arr := w.Array(name, len(values))
	for _, v := range values {
		arr.String(v)
	}
}

func readStrings(r model.FieldReader, name string) []string {
	arr, ok := r.Array(name)
	if !ok {
		return nil
	}
	out := make([]string, arr.Len())
	for i := range out {
		out[i] = arr.String(i)
	}
	return out
}

// MountOperations registers every operation with the RBAC resource and action it needs and the type of its arguments.
func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListProducts, m.opListProducts).Requires(resProduct, model.Read).Accepts(&ListProductsArgs{})
	reg.Operation(OpGetProduct, m.opGetProduct).Requires(resProduct, model.Read).Accepts(&IdArgs{})
	reg.Operation(OpCreateProduct, m.opCreateProduct).Requires(resProduct, model.Create).Accepts(&PharmacyProduct{})
	reg.Operation(OpUpdateProduct, m.opUpdateProduct).Requires(resProduct, model.Update).Accepts(&UpdateProductArgs{})
	reg.Operation(OpUpsertProduct, m.opUpsertProduct).Requires(resProduct, model.Create|model.Update).Accepts(&UpsertProductArgs{})
	reg.Operation(OpDeactivateProduct, m.opDeactivateProduct).Requires(resProduct, model.Update).Accepts(&IdArgs{})
	reg.Operation(OpDeleteProduct, m.opDeleteProduct).Requires(resProduct, model.Delete).Accepts(&DeleteIdsArgs{})

	reg.Operation(OpListLocations, m.opListLocations).Requires(resLocation, model.Read).Accepts(&ListLocationsArgs{})
	reg.Operation(OpUpsertLocation, m.opUpsertLocation).Requires(resLocation, model.Create|model.Update).Accepts(&UpsertLocationArgs{})
	reg.Operation(OpUpdateLocation, m.opUpdateLocation).Requires(resLocation, model.Update).Accepts(&UpdateLocationArgs{})
	reg.Operation(OpDeleteLocation, m.opDeleteLocation).Requires(resLocation, model.Delete).Accepts(&DeleteIdsArgs{})

	reg.Operation(OpListBatches, m.opListBatches).Requires(resBatch, model.Read).Accepts(&ListBatchesArgs{})
	reg.Operation(OpReceiveStock, m.opReceiveStock).Requires(resMovement, model.Create).Accepts(&ReceiveStockArgs{})
	reg.Operation(OpRegisterEntry, m.opRegisterEntry).Requires(resMovement, model.Create).Accepts(&RegisterEntryArgs{})
	reg.Operation(OpRegisterOutput, m.opRegisterOutput).Requires(resMovement, model.Create).Accepts(&RegisterOutputArgs{})
	reg.Operation(OpTransferStock, m.opTransferStock).Requires(resMovement, model.Create).Accepts(&TransferStockArgs{})
	reg.Operation(OpListStock, m.opListStock).Requires(resLevel, model.Read).Accepts(&ListStockArgs{})
	reg.Operation(OpListMovements, m.opListMovements).Requires(resMovement, model.Read).Accepts(&ListMovementsArgs{})
	reg.Operation(OpListExpiring, m.opListExpiring).Requires(resLevel, model.Read).Accepts(&ListExpiringArgs{})
	reg.Operation(OpListLowStock, m.opListLowStock).Requires(resLevel, model.Read).Accepts(&ListLowStockArgs{})
}

var _ router.OperationModule = (*Module)(nil)

// ---- handlers: decode, call the method, encode; every failure goes through writeError ----

// answer encodes v, or answers the failure.
func answer(ctx router.Context, v model.Encodable, err error) {
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(v); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opListProducts(ctx router.Context) {
	var args ListProductsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListProducts(args)
	list := make(PharmacyProductList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}

func (m *Module) opGetProduct(ctx router.Context) {
	var args IdArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	p, err := m.GetProduct(args.TenantId, args.Id)
	answer(ctx, &p, err)
}

func (m *Module) opCreateProduct(ctx router.Context) {
	var p PharmacyProduct
	if err := ctx.Decode(&p); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	out, err := m.CreateProduct(p)
	answer(ctx, &out, err)
}

func (m *Module) opUpdateProduct(ctx router.Context) {
	var args UpdateProductArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	if err := m.PatchProducts(args.Ids, args.Fields, args.Record); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opUpsertProduct(ctx router.Context) {
	var args UpsertProductArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	var last PharmacyProduct
	for _, rec := range args.Records {
		out, err := m.UpsertProduct(rec)
		if err != nil {
			writeError(ctx, err)
			return
		}
		last = out
	}
	answer(ctx, &last, nil)
}

func (m *Module) opDeactivateProduct(ctx router.Context) {
	var args IdArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if err := m.DeactivateProduct(args.TenantId, args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opDeleteProduct(ctx router.Context) {
	var args DeleteIdsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	if err := m.DeleteProducts(args.Ids); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListLocations(ctx router.Context) {
	var args ListLocationsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListLocations(args)
	list := make(PharmacyLocationList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}

func (m *Module) opUpsertLocation(ctx router.Context) {
	var args UpsertLocationArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	var last PharmacyLocation
	for _, rec := range args.Records {
		out, err := m.UpsertLocation(rec)
		if err != nil {
			writeError(ctx, err)
			return
		}
		last = out
	}
	answer(ctx, &last, nil)
}

func (m *Module) opUpdateLocation(ctx router.Context) {
	var args UpdateLocationArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	if err := m.PatchLocations(args.Ids, args.Fields, args.Record); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opDeleteLocation(ctx router.Context) {
	var args DeleteIdsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	if err := m.DeleteLocations(args.Ids); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListBatches(ctx router.Context) {
	var args ListBatchesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListBatches(args.TenantId, args.ProductId)
	list := make(PharmacyBatchList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}

func (m *Module) opReceiveStock(ctx router.Context) {
	var args ReceiveStockArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	mv, err := m.ReceiveStock(args)
	answer(ctx, &mv, err)
}

func (m *Module) opRegisterEntry(ctx router.Context) {
	var args RegisterEntryArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	mv, err := m.RegisterEntry(args)
	answer(ctx, &mv, err)
}

func movementList(items []PharmacyMovement) PharmacyMovementList {
	list := make(PharmacyMovementList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	return list
}

func (m *Module) opRegisterOutput(ctx router.Context) {
	var args RegisterOutputArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.RegisterOutput(args)
	list := movementList(items)
	answer(ctx, &list, err)
}

func (m *Module) opTransferStock(ctx router.Context) {
	var args TransferStockArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.TransferStock(args)
	list := movementList(items)
	answer(ctx, &list, err)
}

func (m *Module) opListStock(ctx router.Context) {
	var args ListStockArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListStock(args)
	list := make(PharmacyStockLevelList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}

func (m *Module) opListMovements(ctx router.Context) {
	var args ListMovementsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListMovements(args)
	list := movementList(items)
	answer(ctx, &list, err)
}

func (m *Module) opListExpiring(ctx router.Context) {
	var args ListExpiringArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListExpiring(args)
	list := make(ExpiringStockList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}

func (m *Module) opListLowStock(ctx router.Context) {
	var args ListLowStockArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListLowStock(args)
	list := make(LowStockList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	answer(ctx, &list, err)
}
