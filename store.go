package pharmacymanager

import "webtyp.com/crud"

func (m *Module) StorePharmacyProduct() crud.Store    { return m.store }
func (m *Module) StorePharmacyLocation() crud.Store   { return m.store }
func (m *Module) StorePharmacyBatch() crud.Store      { return m.store }
func (m *Module) StorePharmacyStockLevel() crud.Store { return m.store }
func (m *Module) StorePharmacyMovement() crud.Store   { return m.store }
func (m *Module) StorePharmacyOperation() crud.Store  { return m.store }
