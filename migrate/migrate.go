// Package migrate reconciles the schema this module owns. It is a package of its own, never called by New, so that nothing
// on a WASM build's import path pulls webtyp.com/ddl: schema work is deploy-time work.
package migrate

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/ddl"
)

// Migrate creates the five tables in foreign-key order: product and location first, then the batch that references the
// product, then the stock level and the movement that reference location and batch.
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	d := ddl.New(conn, ddlCompiler)
	if err := d.CreateTable(&pharmacymanager.PharmacyProduct{}); err != nil {
		return err
	}
	if err := d.CreateTable(&pharmacymanager.PharmacyLocation{}); err != nil {
		return err
	}
	if err := d.CreateTable(&pharmacymanager.PharmacyBatch{}); err != nil {
		return err
	}
	if err := d.CreateTable(&pharmacymanager.PharmacyStockLevel{}); err != nil {
		return err
	}
	return d.CreateTable(&pharmacymanager.PharmacyMovement{})
}
