package tests

import (
	"testing"

	"webtyp.com/view"
	"webtyp.com/view/conformance"
	"webtyp.com/view/mock"
)

// TestViewConformance ejecuta el comportamiento compartido de un view.Presenter contra el motor en el que se construyen las vistas de este módulo.
func TestViewConformance(t *testing.T) {
	conformance.Run(t, conformance.Factory{
		New: func(t *testing.T, p view.Presenter) conformance.Driver {
			r := mock.New(p)
			return conformance.Driver{
				Mount: r.Mount, Labels: r.Labels, Select: r.Select, SetField: r.SetField, Save: r.Save, Delete: r.Delete,
				New: r.New, Edit: r.Edit, Cancel: r.Cancel, FocusedFieldID: r.FocusedFieldID, ActionLabels: r.ActionLabels,
				ActionEnabled: r.ActionEnabled, ClickAction: r.ClickAction, ConfirmAction: r.ConfirmAction,
			}
		},
	})
}
