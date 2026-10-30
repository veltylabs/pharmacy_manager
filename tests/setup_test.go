package tests

import (
	"testing"

	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/fmt"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	"webtyp.com/time"
)

const testTenant = "tenant-1"

// neverExpires significa "no caduca" en este módulo: un lote de farmacia siempre tiene fecha de caducidad (regla del plan),
// así que el lote de la suite sin caducidad (0) se recibe con una fecha mucho más allá de cualquier otra, lo que lo ordena último como la
// suite espera; Lots() lo traduce de vuelta a 0.
const neverExpires time.Unix = 4102444800 // 2100-01-01 UTC

// testIDGen es el doble de prueba de model.IDGenerator: un módulo nunca construye su propio generador.
type testIDGen struct{ n int }

func (g *testIDGen) NewID() string {
	g.n++
	return "id-" + fmt.Convert(g.n).String()
}

// newModule es un Module sobre una base de datos en memoria, sin eventos.
func newModule(t *testing.T) *pharmacymanager.Module {
	t.Helper()
	m, err := pharmacymanager.New(orm.New(mem.New()), pharmacymanager.Deps{IDs: &testIDGen{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// newProduct crea un producto activo, gravado, a la venta (precio 1000) y falla el test si no puede.
func newProduct(t *testing.T, m *pharmacymanager.Module, sku string) pharmacymanager.PharmacyProduct {
	t.Helper()
	p, err := m.CreateProduct(pharmacymanager.PharmacyProduct{
		TenantId: testTenant, Sku: sku, Name: "Product " + sku, Unit: pharmacymanager.UnitBox,
		TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: 1000, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProduct(%s): %v", sku, err)
	}
	return p
}

// newLocation crea una ubicación activa y falla el test si no puede.
func newLocation(t *testing.T, m *pharmacymanager.Module, name string) pharmacymanager.PharmacyLocation {
	t.Helper()
	l, err := m.CreateLocation(pharmacymanager.PharmacyLocation{TenantId: testTenant, Name: name, IsActive: true})
	if err != nil {
		t.Fatalf("CreateLocation(%s): %v", name, err)
	}
	return l
}
