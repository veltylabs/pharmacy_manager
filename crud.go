package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/time"
)

// locationColumns son las columnas que cambian al actualizar una ubicación completa.
var locationColumns = []string{PharmacyLocation_.Name, PharmacyLocation_.IsSalesPoint, PharmacyLocation_.IsActive, PharmacyLocation_.UpdatedAt}

// updateProductTx guarda el producto p sobre el guardado existing: es lo que comparten UpdateProduct y PatchProducts.
func updateProductTx(tx crud.Tx, existing, p *PharmacyProduct) error {
	p.UpdatedAt = time.NowNano()
	if err := checkProduct(p); err != nil {
		return err
	}
	if err := p.Validate(model.ActionUpdate); err != nil {
		return err
	}
	if p.Sku != existing.Sku {
		if err := checkSkuFree(tx, p.Sku); err != nil {
			return err
		}
	}
	return tx.UpdateFields(p, p.Id, productColumns)
}

// updateLocationTx es updateProductTx de las ubicaciones.
func updateLocationTx(tx crud.Tx, existing, l *PharmacyLocation) error {
	l.UpdatedAt = time.NowNano()
	if err := l.Validate(model.ActionUpdate); err != nil {
		return err
	}
	if l.Name != existing.Name {
		if err := checkLocationNameFree(tx, l.Name); err != nil {
			return err
		}
	}
	return tx.UpdateFields(l, l.Id, locationColumns)
}

func (m *Module) UpdateProduct(p PharmacyProduct) (PharmacyProduct, error) {
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		existing, err := getProduct(tx, p.Id)
		if err != nil {
			return err
		}
		return updateProductTx(tx, &existing, &p)
	})
	if err != nil {
		return PharmacyProduct{}, err
	}
	return p, nil
}

func (m *Module) UpsertProduct(p PharmacyProduct) (PharmacyProduct, error) {
	if p.Id == "" {
		return m.CreateProduct(p)
	}
	store, err := m.storeFor(p.TenantId)
	if err != nil {
		return PharmacyProduct{}, err
	}
	if _, err := getProduct(store, p.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateProduct(p)
	} else if err != nil {
		return PharmacyProduct{}, err
	}
	return m.UpdateProduct(p)
}

func (m *Module) PatchProducts(ids, fields []string, rec PharmacyProduct) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			current, err := getProduct(tx, id)
			if err != nil {
				return err
			}
			existing := current
			if err := applyProductFields(&current, &rec, fields); err != nil {
				return err
			}
			if err := updateProductTx(tx, &existing, &current); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) DeactivateProduct(tenantId, id string) error {
	store, err := m.storeFor(tenantId)
	if err != nil {
		return err
	}
	return store.Atomic(func(tx crud.Tx) error {
		p, err := getProduct(tx, id)
		if err != nil {
			return err
		}
		p.IsActive = false
		p.UpdatedAt = time.NowNano()
		return tx.UpdateFields(&p, p.Id, []string{PharmacyProduct_.IsActive, PharmacyProduct_.UpdatedAt})
	})
}

func (m *Module) DeleteProducts(ids []string) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			p, err := getProduct(tx, id)
			if err != nil {
				return err
			}
			if err := checkProductDeletable(tx, p.Id); err != nil {
				return err
			}
			if err := tx.Delete(&p, p.Id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) UpdateLocation(l PharmacyLocation) (PharmacyLocation, error) {
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	err = store.Atomic(func(tx crud.Tx) error {
		existing, err := getLocation(tx, l.Id)
		if err != nil {
			return err
		}
		return updateLocationTx(tx, &existing, &l)
	})
	if err != nil {
		return PharmacyLocation{}, err
	}
	return l, nil
}

func (m *Module) UpsertLocation(l PharmacyLocation) (PharmacyLocation, error) {
	if l.Id == "" {
		return m.CreateLocation(l)
	}
	store, err := m.storeFor(l.TenantId)
	if err != nil {
		return PharmacyLocation{}, err
	}
	if _, err := getLocation(store, l.Id); model.KindOf(err) == model.ErrorNotFound {
		return m.CreateLocation(l)
	} else if err != nil {
		return PharmacyLocation{}, err
	}
	return m.UpdateLocation(l)
}

func (m *Module) PatchLocations(ids, fields []string, rec PharmacyLocation) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			current, err := getLocation(tx, id)
			if err != nil {
				return err
			}
			existing := current
			if err := applyLocationFields(&current, &rec, fields); err != nil {
				return err
			}
			if err := updateLocationTx(tx, &existing, &current); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *Module) DeleteLocations(ids []string) error {
	return m.store.Atomic(func(tx crud.Tx) error {
		for _, id := range ids {
			l, err := getLocation(tx, id)
			if err != nil {
				return err
			}
			if err := checkLocationDeletable(tx, l.Id); err != nil {
				return err
			}
			if err := tx.Delete(&l, l.Id); err != nil {
				return err
			}
		}
		return nil
	})
}
