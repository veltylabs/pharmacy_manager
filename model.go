package pharmacymanager

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// ModelName is the wire and RBAC prefix of the module: operations are called "<ModelName>.<op>".
const ModelName = "pharmacy"

// Units a product is counted in. The select of the form takes its options from these (ui/browser.go, SetOptions).
const (
	UnitPiece  = "PIECE"
	UnitBox    = "BOX"
	UnitTablet = "TABLET"
	UnitMl     = "ML"
	UnitGram   = "GRAM"
)

// Tax treatment of a product. The rate (19 %) is the law's and belongs to the accounting configuration, not here.
const (
	TaxTaxed  = "TAXED"
	TaxExempt = "EXEMPT"
)

// PharmacyProductModel is a medicine or article the pharmacy stocks. Prices are whole Chilean pesos, tax included.
var PharmacyProductModel = model.Definition{
	Name: "pharmacy_product",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "sku", Type: input.Text(), Label: "SKU", NotNull: true, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 50}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Letters: true, Numbers: true, Spaces: true, Tilde: true, Extra: []rune{'.', ',', '(', ')', '-', '/', '%', '+'}, Minimum: 1, Maximum: 255}},
		{Name: "unit", Type: input.Select(), Label: "Unit", NotNull: true},
		{Name: "tax_treatment", Type: input.Select(), Label: "Tax treatment", NotNull: true},
		{Name: "for_sale", Type: input.Checkbox(), Label: "For sale"},
		{Name: "price", Type: input.Number(), Label: "Price", OmitEmpty: true},
		{Name: "controlled", Type: input.Checkbox(), Label: "Controlled"},
		{Name: "barcode", Type: input.Text(), Label: "Barcode", OmitEmpty: true, Permitted: model.Permitted{Numbers: true, Minimum: 8, Maximum: 14}},
		{Name: "registry_number", Type: input.Text(), Label: "Health registry number", OmitEmpty: true, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-', '/'}, Minimum: 1, Maximum: 30}},
		{Name: "min_quantity", Type: input.Decimal(), Label: "Minimum quantity", OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active"},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

// PharmacyLocationModel is a place stock is kept: a storeroom, the counter.
var PharmacyLocationModel = model.Definition{
	Name: "pharmacy_location",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Letters: true, Numbers: true, Spaces: true, Tilde: true, Minimum: 1, Maximum: 100}},
		{Name: "is_sales_point", Type: input.Checkbox(), Label: "Sales point"},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active"},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

// PharmacyBatchModel is one lot of one product. expiration_date and unit_cost are Unix seconds UTC and whole pesos per unit.
var PharmacyBatchModel = model.Definition{
	Name: "pharmacy_batch",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "product_id", Type: model.Text(), Ref: &PharmacyProductModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "batch_number", Type: model.Text(), NotNull: true},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "unit_cost", Type: model.Int(), OmitEmpty: true},
		{Name: "is_active", Type: model.Bool()},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

// PharmacyStockLevelModel is how much of one batch is in one location. One row per (location, batch); revision guards
// concurrent updates.
var PharmacyStockLevelModel = model.Definition{
	Name: "pharmacy_stock_level",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "location_id", Type: model.Text(), Ref: &PharmacyLocationModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "batch_id", Type: model.Text(), Ref: &PharmacyBatchModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "quantity", Type: model.Float()},
		{Name: "revision", Type: model.Int()},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true},
	},
}

// PharmacyMovementModel is one append-only entry of the stock ledger. quantity is always > 0; the direction comes from
// movement_type (webtyp.com/stock). There is no update and no delete: a mistake is corrected by a new movement.
var PharmacyMovementModel = model.Definition{
	Name: "pharmacy_movement",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "batch_id", Type: model.Text(), Ref: &PharmacyBatchModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "location_id", Type: model.Text(), Ref: &PharmacyLocationModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "movement_type", Type: model.Text(), NotNull: true},
		{Name: "quantity", Type: model.Float(), NotNull: true},
		{Name: "source_document_type", Type: model.Text(), OmitEmpty: true},
		{Name: "source_document_id", Type: model.Text(), OmitEmpty: true},
		{Name: "note", Type: model.Text(), OmitEmpty: true},
		{Name: "created_at", Type: model.NanoTime()},
		{Name: "created_by", Type: model.Text(), OmitEmpty: true},
	},
}

// Arguments of the operations. Each has tenant_id first, like service_catalog's.

var ListProductsArgsModel = model.Definition{
	Name: "list_products_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

// IdArgsModel is the argument of every operation that names one record: get, deactivate, delete (product, location).
var IdArgsModel = model.Definition{
	Name: "id_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var ListLocationsArgsModel = model.Definition{
	Name: "list_locations_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
	},
}

var ListBatchesArgsModel = model.Definition{
	Name: "list_batches_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
	},
}

var ReceiveStockArgsModel = model.Definition{
	Name: "receive_stock_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "batch_number", Type: model.Text()},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "unit_cost", Type: model.Int()},
		{Name: "quantity", Type: model.Float()},
		{Name: "source_document_type", Type: model.Text()},
		{Name: "source_document_id", Type: model.Text()},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var RegisterEntryArgsModel = model.Definition{
	Name: "register_entry_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var RegisterOutputArgsModel = model.Definition{
	Name: "register_output_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "movement_type", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "source_document_type", Type: model.Text()},
		{Name: "source_document_id", Type: model.Text()},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var TransferStockArgsModel = model.Definition{
	Name: "transfer_stock_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "from_location_id", Type: model.Text()},
		{Name: "to_location_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var ListStockArgsModel = model.Definition{
	Name: "list_stock_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
	},
}

var ListMovementsArgsModel = model.Definition{
	Name: "list_movements_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "from", Type: model.NanoTime()},
		{Name: "to", Type: model.NanoTime()},
		{Name: "limit", Type: model.Int()},
	},
}

var ListExpiringArgsModel = model.Definition{
	Name: "list_expiring_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "within_days", Type: model.Int()},
	},
}

var ListLowStockArgsModel = model.Definition{
	Name: "list_low_stock_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}

// Results that are not a table.

// ExpiringStockModel is one row of list_expiring: stock that expires within the asked days, or already expired.
var ExpiringStockModel = model.Definition{
	Name: "expiring_stock",
	Fields: model.Fields{
		{Name: "product_id", Type: model.Text()},
		{Name: "product_name", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "batch_number", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "expired", Type: model.Bool()},
	},
}

// LowStockModel is one row of list_low_stock: an active product whose total is at or below its min_quantity.
var LowStockModel = model.Definition{
	Name: "low_stock",
	Fields: model.Fields{
		{Name: "product_id", Type: model.Text()},
		{Name: "product_name", Type: model.Text()},
		{Name: "total", Type: model.Float()},
		{Name: "min_quantity", Type: model.Float()},
	},
}
