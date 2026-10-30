// Package seed carga datos de demo usando los métodos propios del módulo, así cada fila es validada como una real.
package seed

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/fmt"
	"webtyp.com/time"
)

// Data es lo que Load creó, para las demos que se construyen sobre ello.
type Data struct {
	Locations []pharmacymanager.PharmacyLocation
	Products  []pharmacymanager.PharmacyProduct
}

const (
	locStore   = "Bodega"
	locCounter = "Mostrador"

	expiryDays     = 365 // every demo batch expires within a year
	unitCostPesos  = 300
	pricePesos     = 2990
	demoQuantity   = 40
	counterPortion = 10
)

// Load crea dos ubicaciones (el mostrador es punto de venta), tres productos y un lote de cada uno recibido en la tienda.
// Ejemplo: data, err := Load(pm, tenant)
func Load(m *pharmacymanager.Module, tenantID string) (Data, error) {
	data := Data{}
	for _, l := range []pharmacymanager.PharmacyLocation{
		{TenantId: tenantID, Name: locStore, IsActive: true},
		{TenantId: tenantID, Name: locCounter, IsSalesPoint: true, IsActive: true},
	} {
		created, err := m.CreateLocation(l)
		if err != nil {
			return data, fmt.Err("seed: CreateLocation", l.Name, err)
		}
		data.Locations = append(data.Locations, created)
	}

	for _, p := range []pharmacymanager.PharmacyProduct{
		{TenantId: tenantID, Sku: "PARA500", Name: "Paracetamol 500 mg", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxTaxed, ForSale: true, Price: pricePesos, MinQuantity: 10, IsActive: true},
		{TenantId: tenantID, Sku: "SUERO1L", Name: "Suero fisiologico 1 L", Unit: pharmacymanager.UnitPiece, TaxTreatment: pharmacymanager.TaxExempt, ForSale: true, Price: 1500, IsActive: true},
		{TenantId: tenantID, Sku: "MUESTRA", Name: "Muestra medica", Unit: pharmacymanager.UnitBox, TaxTreatment: pharmacymanager.TaxExempt, IsActive: true},
	} {
		created, err := m.CreateProduct(p)
		if err != nil {
			return data, fmt.Err("seed: CreateProduct", p.Sku, err)
		}
		data.Products = append(data.Products, created)
	}

	expiry := time.NowUnix() + expiryDays*time.Day
	for i, p := range data.Products {
		args := pharmacymanager.ReceiveStockArgs{
			TenantId: tenantID, ProductId: p.Id, LocationId: data.Locations[0].Id,
			BatchNumber: "LOT-" + fmt.Convert(i+1).String(), ExpirationDate: expiry, UnitCost: unitCostPesos, Quantity: demoQuantity,
		}
		if _, err := m.ReceiveStock(args); err != nil {
			return data, fmt.Err("seed: ReceiveStock", p.Sku, err)
		}
	}
	if _, err := m.TransferStock(pharmacymanager.TransferStockArgs{
		TenantId: tenantID, ProductId: data.Products[0].Id, FromLocationId: data.Locations[0].Id, ToLocationId: data.Locations[1].Id, Quantity: counterPortion,
	}); err != nil {
		return data, fmt.Err("seed: TransferStock", err)
	}
	return data, nil
}
