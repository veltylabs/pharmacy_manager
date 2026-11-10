package pharmacymanager

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

const ModelName = "pharmacy"
const (
	UnitPiece  = "PIECE"
	UnitBox    = "BOX"
	UnitTablet = "TABLET"
	UnitMl     = "ML"
	UnitGram   = "GRAM"
)
const (
	TaxTaxed  = "TAXED"
	TaxExempt = "EXEMPT"
)

var PharmacyProductModel = model.Definition{
	Name:  "pharmacy_product",
	Allow: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{
		Create: model.AddMyBusinessRule,
		Read:   model.HaveNoBusinessRule,
		Update: model.AddMyBusinessRule,
		Delete: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "sku", Type: input.Text(), Label: "SKU", NotNull: true, Item: model.ItemDescription, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 50}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Item: model.ItemLabel, Permitted: model.Permitted{Letters: true, Numbers: true, Spaces: true, Tilde: true, Extra: []rune{'.', ',', '(', ')', '-', '/', '%', '+'}, Minimum: 1, Maximum: 255}},
		{Name: "unit", Type: input.Select(), Label: "Unit", NotNull: true},
		{Name: "tax_treatment", Type: input.Select(), Label: "Tax treatment", NotNull: true},
		{Name: "for_sale", Type: input.Checkbox(), Label: "For sale"},
		{Name: "price", Type: input.Number(), Label: "Price", OmitEmpty: true},
		{Name: "controlled", Type: input.Checkbox(), Label: "Controlled"},
		{Name: "barcode", Type: input.Text(), Label: "Barcode", OmitEmpty: true, Permitted: model.Permitted{Numbers: true, Minimum: 8, Maximum: 14}},
		{Name: "registry_number", Type: input.Text(), Label: "Health registry number", OmitEmpty: true, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-', '/'}, Minimum: 1, Maximum: 30}},
		{Name: "min_quantity", Type: input.Decimal(), Label: "Minimum quantity", OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active"},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}

var PharmacyLocationModel = model.Definition{
	Name:  "pharmacy_location",
	Allow: model.Create | model.Read | model.Update | model.Delete,
	Rules: model.Rules{
		Create: model.AddMyBusinessRule,
		Read:   model.HaveNoBusinessRule,
		Update: model.AddMyBusinessRule,
		Delete: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Item: model.ItemLabel, Permitted: model.Permitted{Letters: true, Numbers: true, Spaces: true, Tilde: true, Minimum: 1, Maximum: 100}},
		{Name: "is_sales_point", Type: input.Checkbox(), Label: "Sales point"},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active"},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}

var PharmacyBatchModel = model.Definition{
	Name:  "pharmacy_batch",
	Allow: model.Read,
	Rules: model.Rules{
		Read: model.HaveNoBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "product_id", Type: model.Text(), Ref: &PharmacyProductModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "batch_number", Type: model.Text(), NotNull: true},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "unit_cost", Type: model.Int(), OmitEmpty: true, ServerOnly: true},
		{Name: "is_active", Type: model.Bool()},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}

var PharmacyStockLevelModel = model.Definition{
	Name:  "pharmacy_stock_level",
	Allow: model.Read,
	Rules: model.Rules{
		Read: model.HaveNoBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "location_id", Type: model.Text(), Ref: &PharmacyLocationModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "batch_id", Type: model.Text(), Ref: &PharmacyBatchModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "quantity", Type: model.Float(), ServerOnly: true},
		{Name: "revision", Type: model.Int(), ServerOnly: true},
		{Name: "updated_at", Type: model.NanoTime(), OmitEmpty: true, ServerOnly: true},
	},
}

var PharmacyMovementModel = model.Definition{
	Name:  "pharmacy_movement",
	Allow: model.Read,
	Rules: model.Rules{
		Read: model.HaveNoBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "batch_id", Type: model.Text(), Ref: &PharmacyBatchModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "location_id", Type: model.Text(), Ref: &PharmacyLocationModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "movement_type", Type: model.Text(), NotNull: true},
		{Name: "quantity", Type: model.Float(), NotNull: true},
		{Name: "source_document_type", Type: model.Text(), Label: "Document type", OmitEmpty: true},
		{Name: "source_document_id", Type: model.Text(), Label: "Document ID", OmitEmpty: true},
		{Name: "note", Type: model.Text(), OmitEmpty: true},
		{Name: "created_at", Type: model.NanoTime(), ServerOnly: true},
		{Name: "created_by", Type: model.Text(), OmitEmpty: true},
	},
}

const (
	OperationReceive  = "receive"
	OperationEntry    = "entry"
	OperationOutput   = "output"
	OperationTransfer = "transfer"
)

var PharmacyOperationModel = model.Definition{
	Name:  "pharmacy_operation",
	Allow: model.Create | model.Read,
	Rules: model.Rules{
		Create: model.AddMyBusinessRule,
		Read:   model.HaveNoBusinessRule,
	},
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true, ServerOnly: true},
		{Name: "kind", Type: input.Select(), NotNull: true},
		{Name: "product_id", Type: model.Text(), OmitEmpty: true},
		{Name: "location_id", Type: model.Text(), OmitEmpty: true},
		{Name: "batch_number", Type: model.Text(), OmitEmpty: true},
		{Name: "expiration_date", Type: model.UnixTime(), OmitEmpty: true},
		{Name: "unit_cost", Type: model.Int(), OmitEmpty: true},
		{Name: "quantity", Type: model.Float(), OmitEmpty: true},
		{Name: "source_document_type", Type: model.Text(), Label: "Document type", OmitEmpty: true},
		{Name: "source_document_id", Type: model.Text(), Label: "Document ID", OmitEmpty: true},
		{Name: "note", Type: model.Text(), OmitEmpty: true},
		{Name: "created_by", Type: model.Text(), OmitEmpty: true},
		{Name: "batch_id", Type: model.Text(), OmitEmpty: true},
		{Name: "movement_type", Type: model.Text(), OmitEmpty: true},
		{Name: "from_location_id", Type: model.Text(), Label: "Origin location", OmitEmpty: true},
		{Name: "to_location_id", Type: model.Text(), Label: "Target location", OmitEmpty: true},
		{Name: "actor_id", Type: model.Text(), ServerOnly: true},
		{Name: "movement_ids", Type: model.Text(), ServerOnly: true},
		{Name: "created_at", Type: model.NanoTime(), ServerOnly: true},
	},
}

var ReceiveStockArgsModel = model.Definition{
	Name: "receive_stock_args",
	Wire: true,
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "batch_number", Type: model.Text()},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "unit_cost", Type: model.Int()},
		{Name: "quantity", Type: model.Float()},
		{Name: "source_document_type", Type: model.Text(), Label: "Document type"},
		{Name: "source_document_id", Type: model.Text(), Label: "Document ID"},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var RegisterEntryArgsModel = model.Definition{
	Name: "register_entry_args",
	Wire: true,
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
	Wire: true,
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "movement_type", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "source_document_type", Type: model.Text(), Label: "Document type"},
		{Name: "source_document_id", Type: model.Text(), Label: "Document ID"},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var TransferStockArgsModel = model.Definition{
	Name: "transfer_stock_args",
	Wire: true,
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "product_id", Type: model.Text()},
		{Name: "from_location_id", Type: model.Text(), Label: "Origin location"},
		{Name: "to_location_id", Type: model.Text(), Label: "Target location"},
		{Name: "quantity", Type: model.Float()},
		{Name: "note", Type: model.Text()},
		{Name: "created_by", Type: model.Text()},
	},
}

var ExpiringFilterModel = model.Definition{
	Name: "expiring_filter",
	Wire: true,
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text(), OmitEmpty: true},
		{Name: "within_days", Type: model.Int(), OmitEmpty: true},
	},
}

var ExpiringStockModel = model.Definition{
	Name:  "expiring_stock",
	Allow: model.Read,
	Rules: model.Rules{
		Read: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "product_id", Type: model.Text()},
		{Name: "product_name", Type: model.Text()},
		{Name: "batch_id", Type: model.Text()},
		{Name: "batch_number", Type: model.Text()},
		{Name: "location_id", Type: model.Text()},
		{Name: "quantity", Type: model.Float()},
		{Name: "expiration_date", Type: model.UnixTime()},
		{Name: "expired", Type: model.Bool()},
		{Name: "within_days", Type: model.Int(), OmitEmpty: true},
	},
}

var LowStockModel = model.Definition{
	Name:  "low_stock",
	Allow: model.Read,
	Rules: model.Rules{
		Read: model.AddMyBusinessRule,
	},
	Fields: model.Fields{
		{Name: "product_id", Type: model.Text()},
		{Name: "product_name", Type: model.Text()},
		{Name: "total", Type: model.Float()},
		{Name: "min_quantity", Type: model.Float()},
	},
}

var Definition = model.ModuleDefinition{
	Name: "pharmacy_manager",
	Entities: []model.Definition{
		PharmacyProductModel,
		PharmacyLocationModel,
		PharmacyBatchModel,
		PharmacyStockLevelModel,
		PharmacyMovementModel,
		PharmacyOperationModel,
		ExpiringStockModel,
		LowStockModel,
	},
}
