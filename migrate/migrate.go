// Package migrate reconcilia el esquema del que este módulo es dueño. Es un paquete propio, nunca llamado por New, para que nada
// en la ruta de importación de una build WASM arrastre webtyp.com/ddl: el trabajo de esquema es trabajo de deploy-time.
package migrate

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/ddl"
)

// Migrate crea las cinco tablas en orden de clave externa: producto y ubicación primero, luego el lote que referencia al
// producto, luego el nivel de stock y el movimiento que referencian ubicación y lote.
// Ejemplo: err := Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	// Sync y no CreateTable: crea la tabla que falta y además agrega las columnas que un modelo
	// ganó, así un campo nuevo llega a una base ya instalada. Flujo: https://github.com/webtyp/ddl/blob/main/docs/diagrams/SYNC_FLOW.md
	return ddl.New(conn, ddlCompiler).Sync(
		&pharmacymanager.PharmacyProduct{},
		&pharmacymanager.PharmacyLocation{},
		&pharmacymanager.PharmacyBatch{},
		&pharmacymanager.PharmacyStockLevel{},
		&pharmacymanager.PharmacyMovement{},
	)
}
