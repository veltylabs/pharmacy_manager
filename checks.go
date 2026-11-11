package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// checkSkuFree rechaza un SKU que ya tiene otro producto; lo comparten la regla de crear/actualizar producto y los métodos.
// Ejemplo: err := checkSkuFree(tx, "PARA500")   // ErrSkuExists si ya existe
func checkSkuFree(tx crud.Tx, sku string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyProduct{} }, storage.Eq(PharmacyProduct_.Sku, sku))
	if err != nil {
		return err
	}
	if found {
		return ErrSkuExists
	}
	return nil
}

// checkLocationNameFree rechaza un nombre que ya tiene otra ubicación.
// Ejemplo: err := checkLocationNameFree(tx, "Bodega")   // ErrLocationNameExists si ya existe
func checkLocationNameFree(tx crud.Tx, name string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyLocation{} }, storage.Eq(PharmacyLocation_.Name, name))
	if err != nil {
		return err
	}
	if found {
		return ErrLocationNameExists
	}
	return nil
}

// checkProductDeletable rechaza borrar un producto que tiene lotes: el libro mayor conserva su historia.
// Ejemplo: err := checkProductDeletable(tx, "p1")   // ErrProductInUse si tiene lotes
func checkProductDeletable(tx crud.Tx, productId string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, productId))
	if err != nil {
		return err
	}
	if found {
		return ErrProductInUse
	}
	return nil
}

// checkLocationDeletable rechaza borrar una ubicación que alguna fila de nivel de stock referencia.
// Ejemplo: err := checkLocationDeletable(tx, "l1")   // ErrLocationInUse si tiene stock
func checkLocationDeletable(tx crud.Tx, locationId string) error {
	_, found, err := firstRow(tx, func() model.Model { return &PharmacyStockLevel{} }, storage.Eq(PharmacyStockLevel_.LocationId, locationId))
	if err != nil {
		return err
	}
	if found {
		return ErrLocationInUse
	}
	return nil
}

// applyProductFields copia a dst solo los campos de fields tomando sus valores de src; un campo desconocido es un error de
// campo. Lo comparten la regla de actualizar producto y PatchProducts.
// Ejemplo: err := applyProductFields(&current, &rec, []string{"name", "price"})
func applyProductFields(dst, src *PharmacyProduct, fields []string) error {
	for _, f := range fields {
		switch f {
		case PharmacyProduct_.Sku:
			dst.Sku = src.Sku
		case PharmacyProduct_.Name:
			dst.Name = src.Name
		case PharmacyProduct_.Unit:
			dst.Unit = src.Unit
		case PharmacyProduct_.TaxTreatment:
			dst.TaxTreatment = src.TaxTreatment
		case PharmacyProduct_.ForSale:
			dst.ForSale = src.ForSale
		case PharmacyProduct_.Price:
			dst.Price = src.Price
		case PharmacyProduct_.Controlled:
			dst.Controlled = src.Controlled
		case PharmacyProduct_.Barcode:
			dst.Barcode = src.Barcode
		case PharmacyProduct_.RegistryNumber:
			dst.RegistryNumber = src.RegistryNumber
		case PharmacyProduct_.MinQuantity:
			dst.MinQuantity = src.MinQuantity
		case PharmacyProduct_.IsActive:
			dst.IsActive = src.IsActive
		default:
			return &model.FieldError{Field: f, Err: ErrUnknownField}
		}
	}
	return nil
}

// applyLocationFields es applyProductFields de las ubicaciones.
// Ejemplo: err := applyLocationFields(&current, &rec, []string{"name"})
func applyLocationFields(dst, src *PharmacyLocation, fields []string) error {
	for _, f := range fields {
		switch f {
		case PharmacyLocation_.Name:
			dst.Name = src.Name
		case PharmacyLocation_.IsSalesPoint:
			dst.IsSalesPoint = src.IsSalesPoint
		case PharmacyLocation_.IsActive:
			dst.IsActive = src.IsActive
		default:
			return &model.FieldError{Field: f, Err: ErrUnknownField}
		}
	}
	return nil
}
