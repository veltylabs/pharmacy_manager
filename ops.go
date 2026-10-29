package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/rpc"
	"webtyp.com/view"
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

// ---- the wire shapes of the operations the generic view drives (webtyp.com/view sends exactly these) ----

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
	reg.Operation(OpListProducts, m.opListProducts).Requires(model.ResourceOf(&PharmacyProductModel), model.Read).Accepts(&ListProductsArgs{})
	reg.Operation(OpGetProduct, m.opGetProduct).Requires(model.ResourceOf(&PharmacyProductModel), model.Read).Accepts(&IdArgs{})
	reg.Operation(OpCreateProduct, m.opCreateProduct).Requires(model.ResourceOf(&PharmacyProductModel), model.Create).Accepts(&PharmacyProduct{})
	reg.Operation(OpUpdateProduct, m.opUpdateProduct).Requires(model.ResourceOf(&PharmacyProductModel), model.Update).Accepts(&UpdateProductArgs{})
	view.MountSave(reg, OpUpsertProduct, func(p *PharmacyProduct) error { out, err := m.UpsertProduct(*p); if err == nil { *p = out }; return err }).Requires(model.ResourceOf(&PharmacyProductModel), model.Create|model.Update)
	reg.Operation(OpDeactivateProduct, m.opDeactivateProduct).Requires(model.ResourceOf(&PharmacyProductModel), model.Update).Accepts(&IdArgs{})
	view.MountDelete(reg, OpDeleteProduct, m.DeleteProducts).Requires(model.ResourceOf(&PharmacyProductModel), model.Delete)

	reg.Operation(OpListLocations, m.opListLocations).Requires(model.ResourceOf(&PharmacyLocationModel), model.Read).Accepts(&ListLocationsArgs{})
	view.MountSave(reg, OpUpsertLocation, func(l *PharmacyLocation) error { out, err := m.UpsertLocation(*l); if err == nil { *l = out }; return err }).Requires(model.ResourceOf(&PharmacyLocationModel), model.Create|model.Update)
	reg.Operation(OpUpdateLocation, m.opUpdateLocation).Requires(model.ResourceOf(&PharmacyLocationModel), model.Update).Accepts(&UpdateLocationArgs{})
	view.MountDelete(reg, OpDeleteLocation, m.DeleteLocations).Requires(model.ResourceOf(&PharmacyLocationModel), model.Delete)

	reg.Operation(OpListBatches, m.opListBatches).Requires(model.ResourceOf(&PharmacyBatchModel), model.Read).Accepts(&ListBatchesArgs{})
	reg.Operation(OpReceiveStock, m.opReceiveStock).Requires(model.ResourceOf(&PharmacyMovementModel), model.Create).Accepts(&ReceiveStockArgs{})
	reg.Operation(OpRegisterEntry, m.opRegisterEntry).Requires(model.ResourceOf(&PharmacyMovementModel), model.Create).Accepts(&RegisterEntryArgs{})
	reg.Operation(OpRegisterOutput, m.opRegisterOutput).Requires(model.ResourceOf(&PharmacyMovementModel), model.Create).Accepts(&RegisterOutputArgs{})
	reg.Operation(OpTransferStock, m.opTransferStock).Requires(model.ResourceOf(&PharmacyMovementModel), model.Create).Accepts(&TransferStockArgs{})
	reg.Operation(OpListStock, m.opListStock).Requires(model.ResourceOf(&PharmacyStockLevelModel), model.Read).Accepts(&ListStockArgs{})
	reg.Operation(OpListMovements, m.opListMovements).Requires(model.ResourceOf(&PharmacyMovementModel), model.Read).Accepts(&ListMovementsArgs{})
	reg.Operation(OpListExpiring, m.opListExpiring).Requires(model.ResourceOf(&PharmacyStockLevelModel), model.Read).Accepts(&ListExpiringArgs{})
	reg.Operation(OpListLowStock, m.opListLowStock).Requires(model.ResourceOf(&PharmacyStockLevelModel), model.Read).Accepts(&ListLowStockArgs{})
}

var _ router.OperationModule = (*Module)(nil)

// ---- handlers: decode, call the method, encode; every failure goes through writeError ----

// answer encodes v, or answers the failure.

func (m *Module) opListProducts(ctx router.Context) {
	var args ListProductsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListProducts(args)
	list := make(PharmacyProductList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opGetProduct(ctx router.Context) {
	var args IdArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	p, err := m.GetProduct(args.TenantId, args.Id)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&p); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opCreateProduct(ctx router.Context) {
	var p PharmacyProduct
	if err := ctx.Decode(&p); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	out, err := m.CreateProduct(p)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opUpdateProduct(ctx router.Context) {
	var args UpdateProductArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if len(args.Ids) == 0 {
		rpc.WriteError(ctx, ErrNoIds)
		return
	}
	if err := m.PatchProducts(args.Ids, args.Fields, args.Record); err != nil {
		rpc.WriteError(ctx, err)
		return
	}
}

func (m *Module) opDeactivateProduct(ctx router.Context) {
	var args IdArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if err := m.DeactivateProduct(args.TenantId, args.Id); err != nil {
		rpc.WriteError(ctx, err)
		return
	}
}

func (m *Module) opListLocations(ctx router.Context) {
	var args ListLocationsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListLocations(args)
	list := make(PharmacyLocationList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opUpdateLocation(ctx router.Context) {
	var args UpdateLocationArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	if len(args.Ids) == 0 {
		rpc.WriteError(ctx, ErrNoIds)
		return
	}
	if err := m.PatchLocations(args.Ids, args.Fields, args.Record); err != nil {
		rpc.WriteError(ctx, err)
		return
	}
}

func (m *Module) opListBatches(ctx router.Context) {
	var args ListBatchesArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListBatches(args.TenantId, args.ProductId)
	list := make(PharmacyBatchList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opReceiveStock(ctx router.Context) {
	var args ReceiveStockArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	mv, err := m.ReceiveStock(args)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&mv); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opRegisterEntry(ctx router.Context) {
	var args RegisterEntryArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	mv, err := m.RegisterEntry(args)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&mv); err != nil {
		rpc.WriteError(ctx, err)
	}
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
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.RegisterOutput(args)
	list := movementList(items)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opTransferStock(ctx router.Context) {
	var args TransferStockArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.TransferStock(args)
	list := movementList(items)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opListStock(ctx router.Context) {
	var args ListStockArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListStock(args)
	list := make(PharmacyStockLevelList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opListMovements(ctx router.Context) {
	var args ListMovementsArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListMovements(args)
	list := movementList(items)
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opListExpiring(ctx router.Context) {
	var args ListExpiringArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListExpiring(args)
	list := make(ExpiringStockList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}

func (m *Module) opListLowStock(ctx router.Context) {
	var args ListLowStockArgs
	if err := ctx.Decode(&args); err != nil {
		rpc.WriteError(ctx, ErrCannotRead)
		return
	}
	items, err := m.ListLowStock(args)
	list := make(LowStockList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err != nil {
		rpc.WriteError(ctx, err)
		return
	}
	if err := ctx.Encode(&list); err != nil {
		rpc.WriteError(ctx, err)
	}
}
