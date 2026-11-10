package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/storage"
)

func (m *Module) GetProduct(tenantId, id string) (PharmacyProduct, error) {
	return PharmacyProduct{}, ErrProductNotFound
}
func (m *Module) GetLocation(tenantId, id string) (PharmacyLocation, error) {
	return PharmacyLocation{}, ErrLocationNotFound
}
func (m *Module) GetBatch(tenantId, id string) (PharmacyBatch, error) {
	return PharmacyBatch{}, ErrBatchNotFound
}
func (m *Module) ListProducts(tenantId string, activeOnly bool, limit, offset int) ([]PharmacyProduct, error) {
	return nil, nil
}
func (m *Module) ListLocations(tenantId string, activeOnly bool) ([]PharmacyLocation, error) {
	return nil, nil
}
func (m *Module) ListBatches(tenantId, productId string) ([]PharmacyBatch, error) {
	return nil, nil
}
func (m *Module) ListStock(tenantId string, productId, locationId string) ([]PharmacyStockLevel, error) {
	return nil, nil
}
func (m *Module) ListMovements(tenantId, productId, batchId string, from, to int64, limit int) ([]PharmacyMovement, error) {
	return nil, nil
}
