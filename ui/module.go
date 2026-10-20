package ui

import "webtyp.com/svg"

// ID is the identity of the module's screen: the RBAC prefix on the server and the route of the navigation on the client.
const ID = "pharmacy"

// Label is the text of the navigation entry.
const Label = "Farmacia"

const (
	IconPharmacy  = svg.Icon("pharmacy")
	IconProducts  = svg.Icon("pharmacy_products")
	IconLocations = svg.Icon("pharmacy_locations")
)
