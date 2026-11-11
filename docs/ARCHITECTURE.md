# Architecture

## 1. Visión general en un minuto
Este módulo encapsula todo el ciclo de vida de la farmacia, manejando los lotes para respetar caducidades con precisión. Las operaciones no generan transacciones contables aisladas directamente sino que propagan "hechos" para subsistemas a nivel de infraestructura.

## 2. Piezas y estructura
| Componente | Archivos |
|---|---|
| Dominio | `definition.go`, `checks.go`, `stock_tx.go`, `rule_*.go` |
| Infraestructura | `module.go`, `crud.go`, `queries.go`, `reports.go`, `alerts.go`, `store.go`, `rows.go`, `view.go` |
| Vistas y Demo | `ui/`, `seed/` |

## 3. Flujos paso a paso
El flujo general para mover mercancía recae en transacciones atómicas.
- Entrada: (Recepción, Entradas) validan el lote y registran movimiento positivo.
- Salida: Primero-en-caducar-primero-en-salir iterando hasta cubrir la demanda y escribiendo decrementos.
- Transferencia: Traslado entre ubicaciones, genera salidas y entradas emparejadas.
Diagrama completo en [diagrams/STOCK_OPERATION.md](diagrams/STOCK_OPERATION.md).

## 4. Contratos y límites de seguridad
El stock nunca entra en negativo, respaldado por `webtyp.com/stock`.
Un lote caducado nunca se vende.
El sistema comparte rutinas de verificación entre validadores explícitos (`checks.go`) y lógicas transaccionales operativas (`stock_tx.go`).
La base de datos asegura confinamiento mediante el `tenant_id`.
