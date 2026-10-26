---
PLAN: "feat: migrate ui to view.Screen and drop layout dependency"
TAG: v0.1.35
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 3041833613504867010
PR: https://github.com/veltylabs/pharmacy_manager/pull/5
REVIEWED_COMMIT: 525cc044f96d60f40ea0e439e523e3fba7427132
---

> Read AGENTS.md at the repo root before any change.

# Migración de UI a view.Screen y eliminación de dependencia webtyp/layout

## Ecosystem search

- `webtyp.com/view` (`v0.6.45`): define el contrato de pantalla declarativa `view.Screen`, `view.Presenter`, `view.WithDescription`, `view.WithTitle`, y el validador `Screen.Validate()`.
- `veltylabs/pharmacy_manager`: actualmente exporta `ui.Browser(...) (platformd.UIModule, error)` importando `webtyp.com/layout/platformd`, `crudview`, forzando a este módulo a acoplarse con librerías de chasis visual y renderizado.

## Design gate

1. **Prior art**: `webtyp/view` v0.6.44 introdujo `view.Screen` como el contrato único para que los módulos declaren qué muestran como datos puros, sin importar renderizadores ni chasis.
2. **Novice-name test**: `ui.Screens(caller router.Caller) []view.Screen` devuelve las pantallas que ofrece este módulo al menú de la aplicación.
3. **Complexity ledger**: Elimina `webtyp.com/layout` (y sus dependencias indirectas) de `go.mod`.
4. **Where it belongs**: `ui.Screens` vive en `github.com/veltylabs/pharmacy_manager/ui`. Los constructores de Presenter viven en la raíz de `pharmacymanager`.
5. **What it deletes**: Borra `ui.Browser` y el require de `webtyp.com/layout` en `go.mod`.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
go get webtyp.com/view@v0.6.45
go get webtyp.com/router@v0.4.8
```

## APIs used

| Symbol | Signature | Tag | File |
|---|---|---|---|
| `view.Screen` | `type Screen struct { ID string; Label lang.Text; Icon string; Permission model.Resource; Presenter Presenter; Lookups []Lookup; Help lang.Text; Children []Screen }` | `webtyp.com/view v0.6.45` | `screen.go:35` |
| `view.Presenter` | `type Presenter interface { ... }` | `webtyp.com/view v0.6.45` | `presenter.go:12` |
| `view.WithTitle` | `func WithTitle(title string) Option` | `webtyp.com/view v0.6.45` | `view.go:124` |
| `view.WithDescription` | `func WithDescription(d lang.Text) Option` | `webtyp.com/view v0.6.45` | `view.go:138` |
| `view.New` | `func New(l Lister, record model.Model, opts ...Option) Presenter` | `webtyp.com/view v0.6.45` | `view.go:147` |
| `router.Caller` | `type Caller interface { Call(op string, args Encodable, out Decodable, done func(error)); Dispatch(op string, args Encodable) }` | `webtyp.com/router v0.4.8` | `router.go` |

## Deletions

- `ui.Browser` en `ui/browser.go`. Criterio: `grep -rn "func.*Browser(" ui/` -> vacío.
- Todo import de `webtyp.com/layout` en el repositorio. Criterio: `grep -rn "webtyp.com/layout" .` -> vacío.

---

## Stage 1: Soportar view.Option en constructores de Presenter

En `view.go`:
Permitir que `NewProductView` y `NewLocationView` reciban `opts ...view.Option` para configurar descripciones u opciones adicionales.

Actualizar `view.go`:

```go
package pharmacymanager

import (
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	titleProducts  = "Products"
	titleLocations = "Locations"
)

func (m *PharmacyProduct) Item() view.Item {
	return view.Item{ID: m.Id, Label: m.Name, Description: m.Sku}
}

func (m *PharmacyLocation) Item() view.Item {
	return view.Item{ID: m.Id, Label: m.Name}
}

// NewProductView builds the Presenter of the products.
func NewProductView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListProducts, Save: OpUpsertProduct, Update: OpUpdateProduct, Delete: OpDeleteProduct},
		func() model.ModelSlice { return &PharmacyProductList{} })
	allOpts := append([]view.Option{view.WithTitle(titleProducts)}, opts...)
	return view.New(b, &PharmacyProduct{}, allOpts...)
}

// NewLocationView builds the Presenter of the locations.
func NewLocationView(caller router.Caller, opts ...view.Option) view.Presenter {
	b := view.NewCallerLister(caller,
		view.Ops{Module: ModelName, List: OpListLocations, Save: OpUpsertLocation, Update: OpUpdateLocation, Delete: OpDeleteLocation},
		func() model.ModelSlice { return &PharmacyLocationList{} })
	allOpts := append([]view.Option{view.WithTitle(titleLocations)}, opts...)
	return view.New(b, &PharmacyLocation{}, allOpts...)
}
```

---

## Stage 2: Declarar ui.Screens sin dependencias de layout

En `ui/browser.go`:
Eliminar `Browser`.
Implementar `Screens(caller router.Caller) []view.Screen` como un grupo raíz con sus 2 pantallas hijas (`products` y `locations`).

Reescribir `ui/browser.go`:

```go
package ui

import (
	pharmacymanager "github.com/veltylabs/pharmacy_manager"
	"webtyp.com/lang"
	"webtyp.com/router"
	"webtyp.com/view"
)

const (
	descPharmacy  lang.Text = "Pharmacy management: catalog of medications, expiration dates and storage locations."
	descProducts  lang.Text = "Medications and pharmaceuticals catalog: presentation, active ingredients, unit and sale conditions."
	descLocations lang.Text = "Storage locations, shelves and warehouses for medications custody."
)

// Screens declara las pantallas que este módulo ofrece al menú de la app.
func Screens(caller router.Caller) []view.Screen {
	return []view.Screen{{
		ID:    ID,
		Label: Label,
		Icon:  string(IconPharmacy),
		Help:  descPharmacy,
		Children: []view.Screen{
			{
				ID:         ID + ".products",
				Label:      "Productos",
				Icon:       string(IconProducts),
				Permission: "pharmacy_product",
				Presenter:  pharmacymanager.NewProductView(caller, view.WithDescription(descProducts)),
			},
			{
				ID:         ID + ".locations",
				Label:      "Ubicaciones",
				Icon:       string(IconLocations),
				Permission: "pharmacy_location",
				Presenter:  pharmacymanager.NewLocationView(caller, view.WithDescription(descLocations)),
			},
		},
	}}
}
```

---

## Stage 3: Actualizar pruebas unitarias

Actualizar las pruebas en `tests/` para verificar `ui.Screens(caller)`:
- `len(screens) == 1`
- `screens[0].Validate() == nil`
- `screens[0].ID == ui.ID`
- `len(screens[0].Children) == 2`
- Cada hijo valida con `c.Validate() == nil` y tiene `Permission != ""`.
- Asegurar que ningún archivo de prueba importe `webtyp.com/layout`.

---

## Stage 4: Limpieza de dependencias en go.mod y verificación

Ejecutar:
```bash
go mod tidy
```

Verificar que:
1. `grep "webtyp.com/layout" go.mod` retorna vacío.
2. `grep "webtyp.com/layout" go.sum` retorna vacío.
3. `grep -rn "webtyp.com/layout" .` retorna vacío.

Ejecutar la suite de pruebas completa:
```bash
gotest
```

---

## Stage 5: Documentación y regla de idioma en veltylabs

1. En `AGENTS.md` y `README.md`, reemplazar referencias a `ui.Browser(...) (platformd.UIModule, error)` por `ui.Screens(caller router.Caller) []view.Screen`.
2. Ejecutar `langc sync .` y verificar 0 errores.

---

## Stages table

| Etapa | Archivos modificados | Qué hace |
|---|---|---|
| Stage 1 | `view.go` | Agrega `opts ...view.Option` a constructores de Presenter |
| Stage 2 | `ui/browser.go` | Exporta `ui.Screens` y elimina imports de `layout` |
| Stage 3 | `tests/` | Valida `ui.Screens` y remueve dependencias de `layout` |
| Stage 4 | `go.mod`, `go.sum` | `go mod tidy` y ejecución de `gotest` |
| Stage 5 | `AGENTS.md`, `README.md`, `lang.json` | Actualiza documentación en español y sincroniza `langc` |

## Delivery check

1. `grep -rn "webtyp.com/layout" .` retorna vacío.
2. `gotest` pasa con 0 fallos de compilación, 0 fallos de tests y cobertura verificada.
3. `langc sync .` pasa sin errores.
4. El PR se abre inmediatamente al pasar estas comprobaciones.

## Executor notes

All tasks completed successfully. Refactored UI module to use view.Screen and dropped layout dependency.
