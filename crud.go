package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// The operations that the generic view drives (save, update by ids and fields, delete by ids) carry no tenant on the wire:
// the records carry theirs, and an id alone names one row, so these helpers resolve a row by id and then work with the
// tenant the row itself stores. Everything that takes a tenant argument elsewhere in this module filters by it.

// productByID returns a product whatever its tenant, so that an operation sent by id alone can find its tenant.
func (m *Module) productByID(id string) (PharmacyProduct, error) {
	var p PharmacyProduct
	got, err := ReadOnePharmacyProduct(m.db.Query(&p).Where(PharmacyProduct_.Id).Eq(id), &p)
	if orm.IsNotFound(err) {
		return PharmacyProduct{}, ErrProductNotFound
	}
	if err != nil {
		return PharmacyProduct{}, err
	}
	return *got, nil
}

func (m *Module) locationByID(id string) (PharmacyLocation, error) {
	var l PharmacyLocation
	got, err := ReadOnePharmacyLocation(m.db.Query(&l).Where(PharmacyLocation_.Id).Eq(id), &l)
	if orm.IsNotFound(err) {
		return PharmacyLocation{}, ErrLocationNotFound
	}
	if err != nil {
		return PharmacyLocation{}, err
	}
	return *got, nil
}

// UpdateProduct replaces a product with the record given: same id and tenant, every field validated again.
func (m *Module) UpdateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	existing, err := getProduct(m.db, p.TenantId, p.Id)
	if err != nil {
		return PharmacyProduct{}, err
	}
	p.UpdatedAt = time.Now()
	if err := checkProduct(&p); err != nil {
		return PharmacyProduct{}, err
	}
	if err := p.Validate(model.ActionUpdate); err != nil {
		return PharmacyProduct{}, ValidationError{Err: err}
	}
	if p.Sku != existing.Sku {
		if _, found, err := m.productBySKU(m.db, p.TenantId, p.Sku); err != nil {
			return PharmacyProduct{}, err
		} else if found {
			return PharmacyProduct{}, ErrSkuExists
		}
	}
	if err := m.db.Update(&p, orm.Eq(PharmacyProduct_.Id, p.Id), orm.Eq(PharmacyProduct_.TenantId, p.TenantId)); err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

// UpsertProduct creates the product when it has no id or the id is unknown, otherwise updates it.
func (m *Module) UpsertProduct(p PharmacyProduct) (PharmacyProduct, error) {
	if p.Id == "" {
		return m.CreateProduct(p)
	}
	if _, err := getProduct(m.db, p.TenantId, p.Id); err == ErrProductNotFound {
		return m.CreateProduct(p)
	} else if err != nil {
		return PharmacyProduct{}, err
	}
	return m.UpdateProduct(p)
}

// PatchProducts applies to the products named by ids only the fields named by fields, taking the values from rec: the
// shape of the view's "update" operation. A field this module does not know is a ValidationError.
func (m *Module) PatchProducts(ids, fields []string, rec PharmacyProduct) error {
	for _, id := range ids {
		current, err := m.productByID(id)
		if err != nil {
			return err
		}
		for _, f := range fields {
			switch f {
			case "sku":
				current.Sku = rec.Sku
			case "name":
				current.Name = rec.Name
			case "unit":
				current.Unit = rec.Unit
			case "tax_treatment":
				current.TaxTreatment = rec.TaxTreatment
			case "for_sale":
				current.ForSale = rec.ForSale
			case "price":
				current.Price = rec.Price
			case "controlled":
				current.Controlled = rec.Controlled
			case "barcode":
				current.Barcode = rec.Barcode
			case "registry_number":
				current.RegistryNumber = rec.RegistryNumber
			case "min_quantity":
				current.MinQuantity = rec.MinQuantity
			case "is_active":
				current.IsActive = rec.IsActive
			default:
				return ValidationError{Err: ErrUnknownField}
			}
		}
		if _, err := m.UpdateProduct(current); err != nil {
			return err
		}
	}
	return nil
}

// DeactivateProduct hides a product from new receipts without touching its history.
func (m *Module) DeactivateProduct(tenantId, id string) error {
	p, err := getProduct(m.db, tenantId, id)
	if err != nil {
		return err
	}
	p.IsActive = false
	p.UpdatedAt = time.Now()
	return m.db.Update(&p, orm.Eq(PharmacyProduct_.Id, p.Id), orm.Eq(PharmacyProduct_.TenantId, p.TenantId))
}

// DeleteProducts removes products by id, each only if it has no batch (ErrProductInUse): the ledger keeps its history.
func (m *Module) DeleteProducts(ids []string) error {
	for _, id := range ids {
		p, err := m.productByID(id)
		if err != nil {
			return err
		}
		batches, err := m.ListBatches(p.TenantId, p.Id)
		if err != nil {
			return err
		}
		if len(batches) > 0 {
			return ErrProductInUse
		}
		if err := m.db.Delete(&p, orm.Eq(PharmacyProduct_.Id, p.Id), orm.Eq(PharmacyProduct_.TenantId, p.TenantId)); err != nil {
			return err
		}
	}
	return nil
}

// UpdateLocation replaces a location with the record given.
func (m *Module) UpdateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	existing, err := getLocation(m.db, l.TenantId, l.Id)
	if err != nil {
		return PharmacyLocation{}, err
	}
	l.UpdatedAt = time.Now()
	if err := l.Validate(model.ActionUpdate); err != nil {
		return PharmacyLocation{}, ValidationError{Err: err}
	}
	if l.Name != existing.Name {
		var proxy PharmacyLocation
		qb := m.db.Query(&proxy).Where(PharmacyLocation_.Name).Eq(l.Name).Where(PharmacyLocation_.TenantId).Eq(l.TenantId)
		if _, err := ReadOnePharmacyLocation(qb, &proxy); err == nil {
			return PharmacyLocation{}, ErrLocationNameExists
		} else if !orm.IsNotFound(err) {
			return PharmacyLocation{}, err
		}
	}
	if err := m.db.Update(&l, orm.Eq(PharmacyLocation_.Id, l.Id), orm.Eq(PharmacyLocation_.TenantId, l.TenantId)); err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

// UpsertLocation creates the location when it has no id or the id is unknown, otherwise updates it.
func (m *Module) UpsertLocation(l PharmacyLocation) (PharmacyLocation, error) {
	if l.Id == "" {
		return m.CreateLocation(l)
	}
	if _, err := getLocation(m.db, l.TenantId, l.Id); err == ErrLocationNotFound {
		return m.CreateLocation(l)
	} else if err != nil {
		return PharmacyLocation{}, err
	}
	return m.UpdateLocation(l)
}

// PatchLocations applies to the locations named by ids only the fields named by fields, taking the values from rec.
func (m *Module) PatchLocations(ids, fields []string, rec PharmacyLocation) error {
	for _, id := range ids {
		current, err := m.locationByID(id)
		if err != nil {
			return err
		}
		for _, f := range fields {
			switch f {
			case "name":
				current.Name = rec.Name
			case "is_sales_point":
				current.IsSalesPoint = rec.IsSalesPoint
			case "is_active":
				current.IsActive = rec.IsActive
			default:
				return ValidationError{Err: ErrUnknownField}
			}
		}
		if _, err := m.UpdateLocation(current); err != nil {
			return err
		}
	}
	return nil
}

// DeleteLocations removes locations by id, each only if no stock level row references it (ErrLocationInUse).
func (m *Module) DeleteLocations(ids []string) error {
	for _, id := range ids {
		l, err := m.locationByID(id)
		if err != nil {
			return err
		}
		var proxy PharmacyStockLevel
		levels, err := ReadAllPharmacyStockLevel(m.db.Query(&proxy).Where(PharmacyStockLevel_.LocationId).Eq(l.Id).Where(PharmacyStockLevel_.TenantId).Eq(l.TenantId))
		if err != nil {
			return err
		}
		if len(levels) > 0 {
			return ErrLocationInUse
		}
		if err := m.db.Delete(&l, orm.Eq(PharmacyLocation_.Id, l.Id), orm.Eq(PharmacyLocation_.TenantId, l.TenantId)); err != nil {
			return err
		}
	}
	return nil
}
