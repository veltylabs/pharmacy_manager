package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/router"
	"webtyp.com/router/mock"
)

// TestMount_EveryOperationDeclaresItsResourceAndAction: an operation without Requires would be unreachable (zero Access is
// guardado) y debe fallar la compilación de la app, así que el módulo debe montar todas ellas con un recurso y una acción.
func TestMount_EveryOperationDeclaresItsResourceAndAction(t *testing.T) {
	var _ router.OperationModule = newModule(t) // the module is an OperationModule

	r := &mock.Router{}
	newModule(t).MountOperations(r)

	wantOps := []string{
		pharmacymanager.OpListProducts, pharmacymanager.OpGetProduct, pharmacymanager.OpCreateProduct, pharmacymanager.OpUpdateProduct,
		pharmacymanager.OpUpsertProduct, pharmacymanager.OpDeactivateProduct, pharmacymanager.OpDeleteProduct,
		pharmacymanager.OpListLocations, pharmacymanager.OpUpsertLocation, pharmacymanager.OpUpdateLocation, pharmacymanager.OpDeleteLocation,
		pharmacymanager.OpListBatches, pharmacymanager.OpReceiveStock, pharmacymanager.OpRegisterEntry, pharmacymanager.OpRegisterOutput,
		pharmacymanager.OpTransferStock, pharmacymanager.OpListStock, pharmacymanager.OpListMovements, pharmacymanager.OpListExpiring,
		pharmacymanager.OpListLowStock,
	}
	routes := r.Routes()
	if len(routes) != len(wantOps) {
		t.Fatalf("%d operations mounted, want %d", len(routes), len(wantOps))
	}
	for _, info := range routes {
		if info.Resource == "" || info.Action == 0 {
			t.Errorf("%s %s must declare a resource and an action, got %q / %v", info.Method, info.Path, info.Resource, info.Action)
		}
	}
}
