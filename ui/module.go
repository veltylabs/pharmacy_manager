package ui

import "webtyp.com/svg"

// ID es la identidad de la pantalla del módulo: el prefijo RBAC en el servidor y la ruta de navegación en el cliente.
// Ejemplo: const id = ID
const ID = "pharmacy"

// Label es el texto de la entrada de navegación.
// Ejemplo: const label = Label
const Label = "Farmacia"

const (
	IconPharmacy  = svg.Icon("pharmacy")
	IconProducts  = svg.Icon("pharmacy_products")
	IconLocations = svg.Icon("pharmacy_locations")
)
