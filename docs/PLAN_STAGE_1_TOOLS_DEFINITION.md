← [Index](PLAN.md) | Next → [Stage 2](PLAN_STAGE_2_MODULE_STORE.md)

# Stage 1 — tools, `definition.go`, `errors.go`, `migrate/`

## 1.1 Tools (run first, exactly)

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go install webtyp.com/definitionc/cmd/definitionc@v0.1.42
export PATH="$PATH:$(go env GOPATH)/bin"
go get webtyp.com/model@v0.2.19 webtyp.com/crud@v0.0.12 webtyp.com/orm@v0.12.33 webtyp.com/storage@v0.2.1 webtyp.com/ddl@v0.0.49 webtyp.com/router@v0.4.22 webtyp.com/definitionc@v0.1.42 webtyp.com/view@v0.6.74 webtyp.com/appkit@v0.2.35
go mod tidy
```

`GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`; remove every `replace` from `go.mod`. **`definitionc` rewrites `go.mod` to older versions every time it runs: after EVERY run repeat the `go get` line and `go mod tidy`.** If a module cannot be fetched, stop and say so in the PR; never invent a hash.

Facts you need (asked before; do not ask again):

- Arguments: `crud.CreateArgs{Records []model.Model}`, `crud.UpdateArgs{IDs []string, Fields []string, Record model.Model}`, `crud.DeleteArgs{IDs []string}`, `crud.ReadArgs{ID, Search, After string, Limit int64, Filter model.Model}`; never a bare entity pointer, a `*List` or a map. Kinds are flat (`model.Pointer`/`List`/`Map` do not exist).
- Inside a rule `tx` is `crud.Tx`: `ReadOne(&rec, id)`, `ReadAll(newRow, crud.Query{Conditions, Order, After, Limit}, onRow)`, `Create`, `UpdateFields(&rec, id, cols)`, `Delete(&rec, id)`. Conditions: `storage.Eq/Neq/Gt/Gte/Lt/Lte/In/Like` (AND only); the tenant is applied by the store: **never add a `tenant_id` condition**; at most `crud.MaxPage` rows per read.
- **Update rules:** `crud` writes `next` after the rule with exactly the columns in `Fields`. A column the rule derives and the caller did not send is written by the rule with `tx.UpdateFields(&row, id, cols)`; a sent column is rewritten by `crud` from `next`, so the rule assigns `next.X = computed` for it.
- **The `id` of a table:** `ValidateInput` always requires a Text primary key unless the field is `ServerOnly`. A table whose rows the server creates: `id` is `OmitEmpty: true, ServerOnly: true` and Create records OMIT it; never send a `ServerOnly` field.
- `NotNull` on an `Int` or `Bool` rejects the zero value: `is_active` (false) must not be `NotNull`.

- **`storage.IsNotFound` does not exist.** A «not found» is `model.KindOf(err) == model.ErrorNotFound` (the helper `notFound(err, ErrX)` of Stage 2 does it).
- **`ModelName` is written by `definitionc`** (`const ModelName` and `func (m *Module) ModelName() string`): delete the hand-written ones (`definition.go` line 8 and `module.go`), or the build says `ModelName redeclared`.
- A read model's `Read` rule has this signature: `func (p *X) Read(a crud.CurrentUser, filter *X, args crud.ReadArgs, m *Module) (model.ModelSlice, error)`: `filter` comes BEFORE `args`.
- No `map` in code that compiles to WASM; no new generics (`page[T]` of `queries.go` stays as it is).

## 1.2 `definition.go` — exact edits (keep every constant and every `Unit*`, `Tax*`)

1. **Delete** `const ModelName`. Keep every `*Args` model that an exported method signature still uses (`ReceiveStockArgs`, `RegisterEntryArgs`, `RegisterOutputArgs`, `TransferStockArgs`, `ListProductsArgs`, `ListLocationsArgs`, `ListStockArgs`, `ListMovementsArgs`, `ListExpiringArgs`, `ListLowStockArgs`) and mark each `Wire: true`. **Delete** only `IdArgsModel` and `ListBatchesArgsModel` (`grep -rn "IdArgs\|ListBatchesArgs" --include=*.go . | grep -v definition` → empty first; if something still uses one, keep it).
2. Each of the five tables: `id` becomes `{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true, ServerOnly: true}`; `tenant_id` and `updated_at` (and `created_at`, `created_by` of the movement) get `ServerOnly: true`.
3. **`PharmacyProductModel`**: `Allow: model.Create | model.Read | model.Update | model.Delete`, `Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule, Update: model.AddMyBusinessRule, Delete: model.AddMyBusinessRule}`. **`PharmacyLocationModel`**: the same.
4. **`PharmacyBatchModel`, `PharmacyStockLevelModel`, `PharmacyMovementModel`**: `Allow: model.Read`, `Rules: model.Rules{Read: model.HaveNoBusinessRule}` (written only by operations).
5. **New table** `PharmacyOperationModel` (Go type `PharmacyOperation`), written exactly, after the movement:

```go
// Tipos de operación de stock: lo que una fila de pharmacy_operation hace al crearse.
const (
	OperationReceive  = "receive"  // entra mercadería nueva (crea o repone un lote)
	OperationEntry    = "entry"    // corrección entrante de un conteo sobre un lote existente
	OperationOutput   = "output"   // venta, uso interno o descarte
	OperationTransfer = "transfer" // mueve stock entre dos ubicaciones
)

// PharmacyOperationModel es una operación de stock: crear una fila la ejecuta (su regla Create corre la lógica de stock
// dentro de la transacción) y deja los ids de los movimientos que produjo.
var PharmacyOperationModel = model.Definition{
	Name:  "pharmacy_operation",
	Allow: model.Create | model.Read,
	Rules: model.Rules{Create: model.AddMyBusinessRule, Read: model.HaveNoBusinessRule},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true, ServerOnly: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "kind", Type: input.Select(), Label: "Kind", NotNull: true},
		{Name: "product_id", Type: model.Text(), NotNull: true, Ref: &PharmacyProductModel, DB: &model.FieldDB{RefColumn: "id"}},
		{Name: "location_id", Type: model.Text(), NotNull: true, Ref: &PharmacyLocationModel, DB: &model.FieldDB{RefColumn: "id"}},
		{Name: "to_location_id", Type: model.Text(), Label: "Target location", OmitEmpty: true},
		{Name: "batch_id", Type: model.Text(), OmitEmpty: true},
		{Name: "batch_number", Type: model.Text(), OmitEmpty: true},
		{Name: "expiration_date", Type: model.UnixTime(), OmitEmpty: true},
		{Name: "unit_cost", Type: model.Int(), OmitEmpty: true},
		{Name: "quantity", Type: model.Float(), OmitEmpty: true},
		{Name: "movement_type", Type: model.Text(), OmitEmpty: true},
		{Name: "source_document_type", Type: model.Text(), Label: "Document type", OmitEmpty: true},
		{Name: "source_document_id", Type: model.Text(), Label: "Document ID", OmitEmpty: true},
		{Name: "note", Type: model.Text(), OmitEmpty: true},
		{Name: "actor_id", Type: model.Text(), OmitEmpty: true, ServerOnly: true},
		{Name: "movement_ids", Type: model.Text(), OmitEmpty: true, ServerOnly: true},
		{Name: "created_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}
```
   Per kind the caller fills: `receive` → `product_id, location_id, batch_number, expiration_date, unit_cost?, quantity, source_document_*?, note?`; `entry` → `product_id, location_id, batch_id, quantity, note?`; `output` → `product_id, location_id, quantity, movement_type, batch_id?, source_document_*?, note?`; `transfer` → `product_id, location_id` (origin), `to_location_id, quantity, note?`.
6. **Read models** — replace `ExpiringStockModel` and `LowStockModel` (no `DB:` marks) with exactly:

```go
// ExpiringStockModel es una fila de «por caducar»: stock que caduca dentro de within_days días (el filtro), o ya caducado.
var ExpiringStockModel = model.Definition{
	Name:  "expiring_stock",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "within_days", Type: model.Int(), OmitEmpty: true}, // filtro: lo que envía quien llama
		{Name: "product_id", Type: model.Text(), OmitEmpty: true},
		{Name: "product_name", Type: model.Text(), OmitEmpty: true},
		{Name: "batch_id", Type: model.Text(), OmitEmpty: true},
		{Name: "batch_number", Type: model.Text(), OmitEmpty: true},
		{Name: "location_id", Type: model.Text(), OmitEmpty: true},
		{Name: "quantity", Type: model.Float(), OmitEmpty: true},
		{Name: "expiration_date", Type: model.UnixTime(), OmitEmpty: true},
		{Name: "expired", Type: model.Bool(), OmitEmpty: true},
	},
}

// LowStockModel es una fila de «stock bajo»: un producto activo cuyo total está en o por debajo de su mínimo.
var LowStockModel = model.Definition{
	Name:  "low_stock",
	Wire:  true,
	Allow: model.Read,
	Rules: model.Rules{Read: model.AddMyBusinessRule},
	Fields: model.Fields{
		{Name: "product_id", Type: model.Text(), OmitEmpty: true},
		{Name: "product_name", Type: model.Text(), OmitEmpty: true},
		{Name: "total", Type: model.Float(), OmitEmpty: true},
		{Name: "min_quantity", Type: model.Float(), OmitEmpty: true},
	},
}
```
   The Go type of `ExpiringStock` gains the field `WithinDays int64`: `ListExpiring` keeps building rows without it.
7. **`Definition`** (new), at the end of the file:

```go
// Definition declara el módulo: el nombre es el prefijo wire y RBAC de siempre ("pharmacy").
var Definition = model.ModuleDefinition{
	Name: "pharmacy",
	Entities: []model.Definition{
		PharmacyProductModel, PharmacyLocationModel, PharmacyBatchModel, PharmacyStockLevelModel, PharmacyMovementModel,
		PharmacyOperationModel, ExpiringStockModel, LowStockModel,
	},
}
```

## 1.3 `errors.go` — add these three (exact text; keep every existing constant; delete none in this stage)

```go
	ErrOperationKind  model.Invalid = "pharmacy: unknown kind of operation"
	ErrFieldNotEditable model.Invalid = "pharmacy: this field cannot be changed"
	ErrTenantRequired model.Invalid = "pharmacy: the tenant is required"
```
(Run `gofmt` on the file: the alignment of the block is gofmt's, not yours.)

## 1.4 `migrate/migrate.go`

Add `&pharmacymanager.PharmacyOperation{}` as the LAST argument of `Sync` (after the movement: it references product and location) and change the comment «las cinco tablas» to «las seis tablas».

`go vet ./...` still fails (the old methods reference removed pieces): expected until Stage 5. Commit and push.
