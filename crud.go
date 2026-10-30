package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/time"
)

// Las operaciones que la vista genérica maneja (save, update por ids y fields, delete por ids) no llevan inquilino en el cable:
// los registros llevan el suyo, y un id solo nombra una fila, por lo que estos ayudantes resuelven una fila por id y luego trabajan con el
// inquilino que la propia fila almacena. Todo lo que toma un argumento de inquilino en otra parte de este módulo filtra por él.

// productByID devuelve un producto sin importar su inquilino, para que una operación enviada solo por id pueda encontrar su inquilino.
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

// UpdateProduct reemplaza un producto con el registro dado: mismo id e inquilino, cada campo validado nuevamente.
// Ejemplo: pm.UpdateProduct(...)
func (m *Module) UpdateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	existing, err := getProduct(m.db, p.TenantId, p.Id)
	if err != nil {
		return PharmacyProduct{}, err
	}
	p.UpdatedAt = time.NowNano()
	if err := checkProduct(&p); err != nil {
		return PharmacyProduct{}, err
	}
	if err := p.Validate(model.ActionUpdate); err != nil {
		return PharmacyProduct{}, err
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

// UpsertProduct crea el producto cuando no tiene id o el id es desconocido, de lo contrario lo actualiza.
// Ejemplo: pm.UpsertProduct(...)
func (m *Module) UpsertProduct(p PharmacyProduct) (PharmacyProduct, error) {
	if p.Id == "" {
		return m.CreateProduct(p)
	}
	if _, err := getProduct(m.db, p.TenantId, p.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateProduct(p)
	} else if err != nil {
		return PharmacyProduct{}, err
	}
	return m.UpdateProduct(p)
}

// PatchProducts aplica a los productos nombrados por ids solo los campos nombrados por fields, tomando los valores de rec: la
// forma de la operación "update" de la vista. Un campo que este módulo no conoce es un ValidationError.
// Ejemplo: pm.PatchProducts(...)
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
				return &model.FieldError{Field: "<unknown>", Err: ErrUnknownField}
			}
		}
		if _, err := m.UpdateProduct(current); err != nil {
			return err
		}
	}
	return nil
}

// DeactivateProduct oculta un producto de nuevos recibos sin tocar su historia.
// Ejemplo: pm.DeactivateProduct(...)
func (m *Module) DeactivateProduct(tenantId, id string) error {
	p, err := getProduct(m.db, tenantId, id)
	if err != nil {
		return err
	}
	p.IsActive = false
	p.UpdatedAt = time.NowNano()
	return m.db.Update(&p, orm.Eq(PharmacyProduct_.Id, p.Id), orm.Eq(PharmacyProduct_.TenantId, p.TenantId))
}

// DeleteProducts elimina productos por id, cada uno solo si no tiene lote (ErrProductInUse): el libro mayor mantiene su historia.
// Ejemplo: pm.DeleteProducts(...)
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

// UpdateLocation reemplaza una ubicación con el registro dado.
// Ejemplo: pm.UpdateLocation(...)
func (m *Module) UpdateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	existing, err := getLocation(m.db, l.TenantId, l.Id)
	if err != nil {
		return PharmacyLocation{}, err
	}
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionUpdate); err != nil {
		return PharmacyLocation{}, err
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

// UpsertLocation crea la ubicación cuando no tiene id o el id es desconocido, de lo contrario la actualiza.
// Ejemplo: pm.UpsertLocation(...)
func (m *Module) UpsertLocation(l PharmacyLocation) (PharmacyLocation, error) {
	if l.Id == "" {
		return m.CreateLocation(l)
	}
	if _, err := getLocation(m.db, l.TenantId, l.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateLocation(l)
	} else if err != nil {
		return PharmacyLocation{}, err
	}
	return m.UpdateLocation(l)
}

// PatchLocations aplica a las ubicaciones nombradas por ids solo los campos nombrados por fields, tomando los valores de rec.
// Ejemplo: pm.PatchLocations(...)
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
				return &model.FieldError{Field: "<unknown>", Err: ErrUnknownField}
			}
		}
		if _, err := m.UpdateLocation(current); err != nil {
			return err
		}
	}
	return nil
}

// DeleteLocations elimina ubicaciones por id, cada una solo si ninguna fila de nivel de stock la referencia (ErrLocationInUse).
// Ejemplo: pm.DeleteLocations(...)
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
