---
PLAN: "feat(ui): describe pharmacy screens with platformd.Describe (H8b)"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 10868670966356070953
PR: https://github.com/veltylabs/pharmacy_manager/pull/4
REVIEWED_COMMIT: 6f74fb644b949d94ab4572583d14b0f441e8c0b3
---

> Read AGENTS.md at the repo root before any change.

# pharmacy_manager: describe screens with platformd.Describe (H8b)

Part of [MODULE_GUIDANCE_MASTER_PLAN.md](https://github.com/webtyp/docs/blob/main/MODULE_GUIDANCE_MASTER_PLAN.md) (H8b):
- Wrap each screen and the group in `ui/browser.go` with `platformd.Describe`.
- English-only keys in Go (`lang.Text`), translated into Spanish in `lang.json`.
- Tests verify each child module (and the group) implements `platformd.Described`.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```

## Environment

```bash
export GOPRIVATE=webtyp.com/*,github.com/veltylabs/*
export GOFLAGS=-mod=mod
```

A PR that does not compile, or whose tests could not run, is a failed delivery. Never pause to ask permission or report progress: when the Delivery check passes, commit, push and open the PR. Stuck for 30 minutes: open the PR anyway listing what fails.

## Stage 1 — Descriptions and platformd.Describe in `ui/browser.go`

Files: `ui/browser.go`, `lang.json`

1. Define description constants (English keys, type `lang.Text`):
   ```go
   const (
   	descPharmacy  lang.Text = "Pharmacy management: catalog of medications, expiration dates and storage locations."
   	descProducts  lang.Text = "Medications and pharmaceuticals catalog: presentation, active ingredients, unit and sale conditions."
   	descLocations lang.Text = "Storage locations, shelves and warehouses for medications custody."
   )
   ```
2. In `Browser`:
   - Wrap `products` module with `platformd.Describe(..., descProducts)`
   - Wrap `locations` module with `platformd.Describe(..., descLocations)`
   - Wrap the `GroupUIModule` with `platformd.Describe(..., descPharmacy)`
3. Run `langc sync .` and ensure `lang.json` includes Spanish translations:
   - `"Pharmacy management: catalog of medications, expiration dates and storage locations."`: `["Gestión de farmacia: catálogo de medicamentos, vencimientos y ubicaciones de almacenamiento."]`
   - `"Medications and pharmaceuticals catalog: presentation, active ingredients, unit and sale conditions."`: `["Catálogo de medicamentos: presentación, principios activos, unidad y condiciones de venta."]`
   - `"Storage locations, shelves and warehouses for medications custody."`: `["Ubicaciones de almacenamiento, estantes y bodegas para custodia de medicamentos."]`

## Stage 2 — Tests

Files: `tests/ui_test.go`

1. In `tests/ui_test.go`, assert that the returned `platformd.UIModule`:
   - Can be cast to `platformd.Described` and has non-empty `Description()`.
   - Its children can be cast to `platformd.Described` and have non-empty `Description()`.
2. Run `gotest ./...` and ensure all tests pass cleanly.

## Stages

| # | Files | Goal |
|---|---|---|
| 1 | `ui/browser.go`, `lang.json` | Add descriptions with `platformd.Describe` and Spanish translations |
| 2 | `tests/ui_test.go` | Assert `platformd.Described` capability and verify `gotest ./...` |
