//go:build browser

package tests

import (
	"testing"

	"github.com/veltylabs/pharmacy_manager/ui"
	"webtyp.com/model"
	"webtyp.com/view"
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

func TestScreens(t *testing.T) {
	caller := &mockCaller{}
	screens := ui.Screens(view.NewRecords().Caller(caller))

	if len(screens) != 1 {
		t.Fatalf("Screens() returned %d screens, want 1", len(screens))
	}

	root := screens[0]
	if err := root.Validate(); err != nil {
		t.Errorf("root screen validation failed: %v", err)
	}
	if root.ID != ui.ID {
		t.Errorf("root screen ID = %q, want %q", root.ID, ui.ID)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root screen has %d children, want 2", len(root.Children))
	}

	for i, child := range root.Children {
		if err := child.Validate(); err != nil {
			t.Errorf("child %d validation failed: %v", i, err)
		}
		if child.Permission == "" {
			t.Errorf("child %d missing Permission", i)
		}
	}
}
