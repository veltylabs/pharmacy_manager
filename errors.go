package pharmacymanager

import (
	"webtyp.com/orm"
	"webtyp.com/router"
)

// domainError is a rule of the module that was not met; every message starts with "pharmacy: ".
type domainError string

func (e domainError) Error() string { return string(e) }

const (
	// 404
	ErrNotFound         domainError = "pharmacy: not found"
	ErrProductNotFound  domainError = "pharmacy: product not found"
	ErrLocationNotFound domainError = "pharmacy: location not found"
	ErrBatchNotFound    domainError = "pharmacy: batch not found"

	// 409
	ErrSkuExists           domainError = "pharmacy: a product with this SKU already exists"
	ErrLocationNameExists  domainError = "pharmacy: a location with this name already exists"
	ErrBatchExists         domainError = "pharmacy: this batch number already exists for the product"
	ErrConflict            domainError = "pharmacy: concurrent update, retry"
	ErrBatchExpiryMismatch domainError = "pharmacy: the batch exists with a different expiry date"
	ErrBatchCostMismatch   domainError = "pharmacy: the batch exists with a different unit cost"

	// 400
	ErrInsufficientStock    domainError = "pharmacy: not enough stock"
	ErrBatchExpired         domainError = "pharmacy: the batch is expired"
	ErrExpiryRequired       domainError = "pharmacy: a batch needs an expiry date"
	ErrSaleNeedsPrice       domainError = "pharmacy: a product for sale needs a price greater than zero"
	ErrNotForSale           domainError = "pharmacy: the product is not for sale"
	ErrTaxTreatmentRequired domainError = "pharmacy: the tax treatment must be TAXED or EXEMPT"
	ErrUnit                 domainError = "pharmacy: unknown unit"
	ErrMovementType         domainError = "pharmacy: this movement type is not allowed here"
	ErrBatchNumber          domainError = "pharmacy: the batch number is required"
	ErrUnitCost             domainError = "pharmacy: the unit cost cannot be negative"
	ErrProductInUse         domainError = "pharmacy: the product has batches"
	ErrLocationInUse        domainError = "pharmacy: the location has stock"
	ErrSameLocation         domainError = "pharmacy: the origin and the destination are the same location"
	ErrProductInactive      domainError = "pharmacy: the product is inactive"
	ErrLocationInactive     domainError = "pharmacy: the location is inactive"
	ErrNoRecords            domainError = "pharmacy: no records to save"
	ErrNoIds                domainError = "pharmacy: no ids given"
	ErrDays                 domainError = "pharmacy: the days cannot be negative"
)

// ValidationError wraps a model validation failure: 400.
type ValidationError struct{ Err error }

func (v ValidationError) Error() string { return v.Err.Error() }

// decodeError is a request body that could not be read: always 400.
type decodeError struct{ err error }

const msgDecode = "pharmacy: cannot read the request: "

func (d decodeError) Error() string { return msgDecode + d.err.Error() }

// writeError answers err with its status and, always, its text. 400 = invalid input · 404 = not found ·
// 409 = conflict · 500 = anything else.
func writeError(ctx router.Context, err error) {
	ctx.WriteStatus(statusOf(err))
	ctx.Write([]byte(err.Error()))
}

func statusOf(err error) int {
	switch e := err.(type) {
	case decodeError, ValidationError:
		return 400
	case domainError:
		switch e {
		case ErrNotFound, ErrProductNotFound, ErrLocationNotFound, ErrBatchNotFound:
			return 404
		case ErrSkuExists, ErrLocationNameExists, ErrBatchExists, ErrConflict, ErrBatchExpiryMismatch, ErrBatchCostMismatch:
			return 409
		default:
			return 400
		}
	}
	if orm.IsNotFound(err) {
		return 404
	}
	return 500
}
