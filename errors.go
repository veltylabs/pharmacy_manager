package pharmacymanager

import (
	"webtyp.com/model"
)

const (
	// 404
	ErrNotFound         model.NotFound = "pharmacy: not found"
	ErrProductNotFound  model.NotFound = "pharmacy: product not found"
	ErrLocationNotFound model.NotFound = "pharmacy: location not found"
	ErrBatchNotFound    model.NotFound = "pharmacy: batch not found"

	// 409
	ErrSkuExists           model.Conflict = "pharmacy: a product with this SKU already exists"
	ErrLocationNameExists  model.Conflict = "pharmacy: a location with this name already exists"
	ErrBatchExists         model.Conflict = "pharmacy: this batch number already exists for the product"
	ErrConflict            model.Conflict = "pharmacy: concurrent update, retry"
	ErrBatchExpiryMismatch model.Conflict = "pharmacy: the batch exists with a different expiry date"
	ErrBatchCostMismatch   model.Conflict = "pharmacy: the batch exists with a different unit cost"

	// 400
	ErrInsufficientStock    model.Invalid = "pharmacy: not enough stock"
	ErrBatchExpired         model.Invalid = "pharmacy: the batch is expired"
	ErrExpiryRequired       model.Invalid = "pharmacy: a batch needs an expiry date"
	ErrSaleNeedsPrice       model.Invalid = "pharmacy: a product for sale needs a price greater than zero"
	ErrNotForSale           model.Invalid = "pharmacy: the product is not for sale"
	ErrTaxTreatmentRequired model.Invalid = "pharmacy: the tax treatment must be TAXED or EXEMPT"
	ErrUnit                 model.Invalid = "pharmacy: unknown unit"
	ErrMovementType         model.Invalid = "pharmacy: this movement type is not allowed here"
	ErrBatchNumber          model.Invalid = "pharmacy: the batch number is required"
	ErrUnitCost             model.Invalid = "pharmacy: the unit cost cannot be negative"
	ErrProductInUse         model.Invalid = "pharmacy: the product has batches"
	ErrLocationInUse        model.Invalid = "pharmacy: the location has stock"
	ErrSameLocation         model.Invalid = "pharmacy: the origin and the destination are the same location"
	ErrProductInactive      model.Invalid = "pharmacy: the product is inactive"
	ErrLocationInactive     model.Invalid = "pharmacy: the location is inactive"
	ErrNoRecords            model.Invalid = "pharmacy: no records to save"
	ErrNoIds                model.Invalid = "pharmacy: no ids given"
	ErrUnknownField         model.Invalid = "pharmacy: unknown field in the update"
	ErrDays                 model.Invalid = "pharmacy: the days cannot be negative"
	ErrCannotRead           model.Invalid = "pharmacy_manager: cannot read the request"
)
