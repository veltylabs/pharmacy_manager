package pharmacymanager

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

// firstRow responde la primera fila que cumple conds y si existe alguna.
//
// Ejemplo: row, found, err := firstRow(tx, func() model.Model { return &PharmacyProduct{} }, storage.Eq(PharmacyProduct_.Sku, "p1"))
func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

// allRows responde todas las filas que cumplen conds, paginando por id porque una lectura trae como máximo crud.MaxPage filas.
//
// Ejemplo: rows, err := allRows(tx, func() model.Model { return &PharmacyBatch{} }, storage.Eq(PharmacyBatch_.ProductId, "p1"))
func allRows(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) ([]model.Model, error) {
	var rows []model.Model
	after := ""
	for {
		var page []model.Model
		err := tx.ReadAll(newRow, crud.Query{Conditions: conds, After: after, Limit: crud.MaxPage},
			func(r model.Model) { page = append(page, r) })
		if err != nil {
			return nil, err
		}
		rows = append(rows, page...)
		if len(page) < crud.MaxPage {
			return rows, nil
		}
		after = keyOf(page[len(page)-1])
	}
}

// keyOf responde el id de una fila de las tablas de este módulo.
func keyOf(r model.Model) string {
	switch row := r.(type) {
	case *PharmacyProduct:
		return row.Id
	case *PharmacyLocation:
		return row.Id
	case *PharmacyBatch:
		return row.Id
	case *PharmacyStockLevel:
		return row.Id
	case *PharmacyMovement:
		return row.Id
	case *PharmacyOperation:
		return row.Id
	}
	return ""
}

// notFound cambia el error «no existe» de un repositorio por `missing` y deja pasar cualquier otro error.
//
// Ejemplo: err = notFound(err, ErrProductNotFound)
func notFound(err error, missing error) error {
	if model.KindOf(err) == model.ErrorNotFound {
		return missing
	}
	return err
}

// containsString dice si list contiene value.
//
// Ejemplo: ok := containsString([]string{"a", "b"}, "b")
func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
