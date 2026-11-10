package migrate

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/ddl"
)

func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).Sync(
		&pharmacymanager.PharmacyProduct{},
		&pharmacymanager.PharmacyLocation{},
		&pharmacymanager.PharmacyBatch{},
		&pharmacymanager.PharmacyStockLevel{},
		&pharmacymanager.PharmacyMovement{},
		&pharmacymanager.PharmacyOperation{},
	)
}
