# Operación de Stock

Contexto: Flujo invocado para el control de inventario y verificado por `tests/rules_test.go`, `tests/output_test.go`, `tests/stock_test.go`.

```mermaid
flowchart TD
    A[Tipo de operación] --> B[Recibir / Entrada]
    A --> C[Salida]
    A --> D[Transferencia]

    B --> E[Verificaciones]
    C --> E
    D --> E

    E -->|ErrInsufficientStock| ERR1[Stock insuficiente]
    E -->|ErrBatchExpired| ERR2[Lote expirado]
    E -->|ErrExpiryRequired| ERR3[Caducidad requerida]
    E -->|ErrSameLocation| ERR4[Misma ubicación de origen y destino]
    E -->|ErrOperationKind| ERR5[Tipo de operación desconocida]

    E --> F[Escribir niveles y registro contable]
    F --> G[Publicar eventos y costos, generar alertas]
```

1. **Determinación del Tipo**: Evalúa si es entrada, salida o traslado.
2. **Validaciones**: Confirma invariantes del negocio previas al movimiento.
3. **Escritura**: Modifica o crea niveles de stock y registra movimientos de manera segura.
4. **Disparadores Secundarios**: Genera la observabilidad posterior al evento (alertas de stock bajo, eventos de sistema, costos).
