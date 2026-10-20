package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"github.com/veltylabs/pharmacy_manager/ui"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
)

// mockCaller is a router.Caller that records the last call and answers it with onCall.
type mockCaller struct {
	lastOp string
	onCall func(op string, args model.Encodable, into model.Decodable) error
}

func (m *mockCaller) Call(op string, args model.Encodable, into model.Decodable, done func(err error)) {
	m.lastOp = op
	if m.onCall != nil {
		done(m.onCall(op, args, into))
		return
	}
	done(nil)
}

func (m *mockCaller) Dispatch(op string, args model.Encodable) { m.lastOp = op }

func TestBrowser_IsAGroupOfProductsAndLocations(t *testing.T) {
	caller := &mockCaller{}
	screen, err := ui.Browser(caller, &testIDGen{}, testTenant)
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}
	if screen.ModelName() != ui.ID || screen.Label() != ui.Label {
		t.Errorf("the group must be %q / %q, got %q / %q", ui.ID, ui.Label, screen.ModelName(), screen.Label())
	}
	group, ok := screen.(platformd.GroupModule)
	if !ok {
		t.Fatalf("expected a platformd.GroupModule, got %T", screen)
	}
	children := group.Children()
	if len(children) != 2 || children[0].Label() != "Productos" || children[1].Label() != "Ubicaciones" {
		t.Fatalf("two children Productos and Ubicaciones expected, got %d", len(children))
	}
	for i, c := range children {
		if _, ok := c.View().(*crudview.CrudView); !ok {
			t.Errorf("child %d must be a *crudview.CrudView, got %T", i, c.View())
		}
	}
}

func TestBrowser_ActivatingLoadsTheProducts(t *testing.T) {
	caller := &mockCaller{}
	screen, err := ui.Browser(caller, &testIDGen{}, testTenant)
	if err != nil {
		t.Fatal(err)
	}
	first := screen.(platformd.GroupModule).Children()[0]
	first.View().(*crudview.CrudView).Init(nil)
	screen.Activate()

	if want := pharmacymanager.ModelName + "." + pharmacymanager.OpListProducts; caller.lastOp != want {
		t.Errorf("activating the group must list the products: last op %q, want %q", caller.lastOp, want)
	}
}
