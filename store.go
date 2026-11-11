package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/orm"
)

// StorePharmacyProduct es el repositorio de los productos.
//
// Ejemplo: store := m.StorePharmacyProduct()
func (m *Module) StorePharmacyProduct() crud.Store { return m.store }

// StorePharmacyLocation es el repositorio de las ubicaciones.
//
// Ejemplo: store := m.StorePharmacyLocation()
func (m *Module) StorePharmacyLocation() crud.Store { return m.store }

// StorePharmacyBatch es el repositorio de los lotes.
//
// Ejemplo: store := m.StorePharmacyBatch()
func (m *Module) StorePharmacyBatch() crud.Store { return m.store }

// StorePharmacyStockLevel es el repositorio de los niveles de stock.
//
// Ejemplo: store := m.StorePharmacyStockLevel()
func (m *Module) StorePharmacyStockLevel() crud.Store { return m.store }

// StorePharmacyMovement es el repositorio de los movimientos.
//
// Ejemplo: store := m.StorePharmacyMovement()
func (m *Module) StorePharmacyMovement() crud.Store { return m.store }

// StorePharmacyOperation es el repositorio de las operaciones de stock.
//
// Ejemplo: store := m.StorePharmacyOperation()
func (m *Module) StorePharmacyOperation() crud.Store { return m.store }

// storeFor es el repositorio de un tenant para los métodos que reciben el tenant como argumento: el del módulo es
// m.store; otro tenant tiene su propio repositorio sellado, que no ve las filas de los demás. Un tenant vacío es un error:
// un repositorio sin tenant vería todas las filas.
//
// Ejemplo: store, err := m.storeFor("tenant-1")
func (m *Module) storeFor(tenantId string) (*orm.Store, error) {
	if tenantId == "" {
		return nil, ErrTenantRequired
	}
	if tenantId == m.tenantID {
		return m.store, nil
	}
	return orm.TenantStore(m.db, tenantId, m.ids), nil
}
