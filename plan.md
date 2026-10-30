1. **Stage 1 — tests first: `tests/migrate_test.go` (new file)**
   - Run `sed -n '66,186p' docs/PLAN.md > tests/migrate_test.go` to create the test file exactly as presented in the `docs/PLAN.md`.
   - Run `gotest`. Ensure that `TestMigrate_NewField_AddsOnlyThatColumn` fails.
   - Run `git add tests/migrate_test.go && git commit -m "tests first"`.

2. **Stage 2 — `migrate/migrate.go`**
   - Run `sed -i '13,26c\\t// Sync y no CreateTable: crea la tabla que falta y además agrega las columnas que un modelo\n\t// ganó, así un campo nuevo llega a una base ya instalada. Flujo: https://github.com/webtyp/ddl/blob/main/docs/diagrams/SYNC_FLOW.md\n\treturn ddl.New(conn, ddlCompiler).Sync(\n\t\t&pharmacymanager.PharmacyProduct{},\n\t\t&pharmacymanager.PharmacyLocation{},\n\t\t&pharmacymanager.PharmacyBatch{},\n\t\t&pharmacymanager.PharmacyStockLevel{},\n\t\t&pharmacymanager.PharmacyMovement{},\n\t)' migrate/migrate.go` to replace the body of `Migrate`.
   - Run `go mod tidy`.
   - Run `gotest` and ensure all tests pass.
   - Run `git add migrate/migrate.go && git commit -m "Migrate with Sync"`.

3. **Stage 3 — docs**
   - Modify `docs/ARCHITECTURE.md`: `sed -i "s/The schema is created by migrate.Migrate(conn, compiler) from the app's migration step, never by New./El esquema se concilia con migrate.Migrate usando ddl.Sync (flujo: https:\/\/github.com\/webtyp\/ddl\/blob\/main\/docs\/diagrams\/SYNC_FLOW.md)./" docs/ARCHITECTURE.md`.
   - Modify `README.md`: `sed -i "s/crea las cinco tablas en orden de clave externa/concilia el esquema usando ddl.Sync (flujo: https:\/\/github.com\/webtyp\/ddl\/blob\/main\/docs\/diagrams\/SYNC_FLOW.md)/" README.md`.
   - Run `git add docs/ARCHITECTURE.md README.md && git commit -m "docs"`.

4. **Stage 4 — language**
   - Create a python script `translate.py` that contains the dictionary mapping english to spanish, and applies them to all go files. Example of translations: `// Source types of the alerts...` -> `// Tipos de origen de las alertas...`, and injects `// Ejemplo:` strings using regex matching on exported functions `func UpsertProduct`, `func NewProductView`, etc. Example mapping:
```python
import os
import re

translations = {
    "// Alert codes this module raises (notify.Alert.Code).": "// Códigos de alerta que este módulo lanza (notify.Alert.Code).",
    "// Source types of the alerts (notify.Ref.Type).": "// Tipos de origen de las alertas (notify.Ref.Type).",
    "// AlertKinds is the catalog of alerts this module raises, for the application's notify.Registry.": "// AlertKinds es el catálogo de alertas que lanza este módulo, para el notify.Registry de la aplicación.",
    "// area is the application's notify.Area code (the same value as Deps.AlertArea).": "// area es el código notify.Area de la aplicación (el mismo valor que Deps.AlertArea).",
    "// raise publishes a when alerts are on; off (no Publisher or no AlertArea) it does nothing.": "// raise publica cuando las alertas están activadas; si no (sin Publisher o sin AlertArea) no hace nada.",
    "// totalProduct is the product's stock over every location and batch.": "// totalProduct es el stock del producto en cada ubicación y lote.",
    "// ModelName is the wire and RBAC prefix of the module: operations are called \"<ModelName>.<op>\".": "// ModelName es el prefijo wire y RBAC del módulo: las operaciones se llaman \"<ModelName>.<op>\".",
    "// Units a product is counted in. The select of the form takes its options from these (ui/browser.go, SetOptions).": "// Unidades en las que se cuenta un producto. El select del formulario toma sus opciones de aquí (ui/browser.go, SetOptions).",
    "// Tax treatment of a product. The rate (19 %) is the law's and belongs to the accounting configuration, not here.": "// Tratamiento fiscal de un producto. La tasa (19 %) es de ley y pertenece a la configuración contable, no aquí.",
    "// PharmacyProductModel is a medicine or article the pharmacy stocks. Prices are whole Chilean pesos, tax included.": "// PharmacyProductModel es una medicina o artículo que la farmacia almacena. Los precios son en pesos chilenos enteros, impuestos incluidos.",
    "// PharmacyLocationModel is a place stock is kept: a storeroom, the counter.": "// PharmacyLocationModel es un lugar donde se mantiene el stock: un almacén, el mostrador.",
    "// PharmacyBatchModel is one lot of one product. expiration_date and unit_cost are Unix seconds UTC and whole pesos per unit.": "// PharmacyBatchModel es un lote de un producto. expiration_date y unit_cost son segundos Unix UTC y pesos enteros por unidad.",
    "// PharmacyStockLevelModel is how much of one batch is in one location. One row per (location, batch); revision guards": "// PharmacyStockLevelModel es cuánto de un lote hay en una ubicación. Una fila por (ubicación, lote); revision guarda",
    "// concurrent updates.": "// actualizaciones concurrentes.",
    "// PharmacyMovementModel is one append-only entry of the stock ledger. quantity is always > 0; the direction comes from": "// PharmacyMovementModel es una entrada solo-adición del libro mayor de stock. quantity es siempre > 0; la dirección viene de",
    "// movement_type (webtyp.com/stock). There is no update and no delete: a mistake is corrected by a new movement.": "// movement_type (webtyp.com/stock). No hay update ni delete: un error se corrige con un nuevo movimiento.",
    "// Arguments of the operations. Each has tenant_id first, like service_catalog's.": "// Argumentos de las operaciones. Cada uno tiene tenant_id primero, como los de service_catalog.",
    "// IdArgsModel is the argument of every operation that names one record: get, deactivate, delete (product, location).": "// IdArgsModel es el argumento de cada operación que nombra un registro: get, deactivate, delete (product, location).",
    "// Results that are not a table.": "// Resultados que no son una tabla.",
    "// ExpiringStockModel is one row of list_expiring: stock that expires within the asked days, or already expired.": "// ExpiringStockModel es una fila de list_expiring: stock que caduca dentro de los días solicitados, o ya caducado.",
    "// LowStockModel is one row of list_low_stock: an active product whose total is at or below its min_quantity.": "// LowStockModel es una fila de list_low_stock: un producto activo cuyo total está en o por debajo de su min_quantity.",
    "// Operation names, as the transport and the view see them: \"<ModelName>.<name>\".": "// Nombres de operación, como el transporte y la vista los ven: \"<ModelName>.<name>\".",
    "// ---- the wire shapes of the operations the generic view drives (webtyp.com/view sends exactly these) ----": "// ---- las formas de cable de las operaciones que la vista genérica maneja (webtyp.com/view envía exactamente estas) ----",
    "// UpdateProductArgs is what view's update sends: the ids, the names of the fields to change and a record with their values.": "// UpdateProductArgs es lo que el update de la vista envía: los ids, los nombres de los campos a cambiar y un registro con sus valores.",
    "// UpdateLocationArgs is the same shape for a location.": "// UpdateLocationArgs es la misma forma para una ubicación.",
    "// MountOperations registers every operation with the RBAC resource and action it needs and the type of its arguments.": "// MountOperations registra cada operación con el recurso RBAC y la acción que necesita y el tipo de sus argumentos.",
    "// ---- handlers: decode, call the method, encode; every failure goes through writeError ----": "// ---- manejadores: decode, call the method, encode; cada falla pasa por writeError ----",
    "// answer encodes v, or answers the failure.": "// answer codifica v, o responde el fallo.",
    "// demoTenantID is the only tenant of this in-browser demo.": "// demoTenantID es el único inquilino de esta demo en el navegador.",
    "// demoUser is the fixed identity the demo shell shows: the demo has no login.": "// demoUser es la identidad fija que muestra el shell de la demo: la demo no tiene login.",
    "// The operations that the generic view drives (save, update by ids and fields, delete by ids) carry no tenant on the wire:": "// Las operaciones que la vista genérica maneja (save, update por ids y fields, delete por ids) no llevan inquilino en el cable:",
    "// the records carry theirs, and an id alone names one row, so these helpers resolve a row by id and then work with the": "// los registros llevan el suyo, y un id solo nombra una fila, por lo que estos ayudantes resuelven una fila por id y luego trabajan con el",
    "// tenant the row itself stores. Everything that takes a tenant argument elsewhere in this module filters by it.": "// inquilino que la propia fila almacena. Todo lo que toma un argumento de inquilino en otra parte de este módulo filtra por él.",
    "// productByID returns a product whatever its tenant, so that an operation sent by id alone can find its tenant.": "// productByID devuelve un producto sin importar su inquilino, para que una operación enviada solo por id pueda encontrar su inquilino.",
    "// UpdateProduct replaces a product with the record given: same id and tenant, every field validated again.": "// UpdateProduct reemplaza un producto con el registro dado: mismo id e inquilino, cada campo validado nuevamente.",
    "// UpsertProduct creates the product when it has no id or the id is unknown, otherwise updates it.": "// UpsertProduct crea el producto cuando no tiene id o el id es desconocido, de lo contrario lo actualiza.",
    "// PatchProducts applies to the products named by ids only the fields named by fields, taking the values from rec: the": "// PatchProducts aplica a los productos nombrados por ids solo los campos nombrados por fields, tomando los valores de rec: la",
    "// shape of the view's \"update\" operation. A field this module does not know is a ValidationError.": "// forma de la operación \"update\" de la vista. Un campo que este módulo no conoce es un ValidationError.",
    "// DeactivateProduct hides a product from new receipts without touching its history.": "// DeactivateProduct oculta un producto de nuevos recibos sin tocar su historia.",
    "// DeleteProducts removes products by id, each only if it has no batch (ErrProductInUse): the ledger keeps its history.": "// DeleteProducts elimina productos por id, cada uno solo si no tiene lote (ErrProductInUse): el libro mayor mantiene su historia.",
    "// UpdateLocation replaces a location with the record given.": "// UpdateLocation reemplaza una ubicación con el registro dado.",
    "// UpsertLocation creates the location when it has no id or the id is unknown, otherwise updates it.": "// UpsertLocation crea la ubicación cuando no tiene id o el id es desconocido, de lo contrario la actualiza.",
    "// PatchLocations applies to the locations named by ids only the fields named by fields, taking the values from rec.": "// PatchLocations aplica a las ubicaciones nombradas por ids solo los campos nombrados por fields, tomando los valores de rec.",
    "// DeleteLocations removes locations by id, each only if no stock level row references it (ErrLocationInUse).": "// DeleteLocations elimina ubicaciones por id, cada una solo si ninguna fila de nivel de stock la referencia (ErrLocationInUse).",
    "// TopicMovementRegistered is published once per stock movement, after the transaction commits. Payload: *PharmacyMovement.": "// TopicMovementRegistered se publica una vez por movimiento de stock, después de que la transacción se confirma. Payload: *PharmacyMovement.",
    "// sourceType is the Ref.Type of the cost facts: the movement is their cause.": "// sourceType es el Ref.Type de los hechos de costo: el movimiento es su causa.",
    "// Deps are what the module needs from the application that composes it.": "// Deps son lo que el módulo necesita de la aplicación que lo compone.",
    "// ---- products and locations (the pattern the rest of the CRUD follows) ----": "// ---- productos y ubicaciones (el patrón que sigue el resto del CRUD) ----",
    "// checkProduct applies the rules of a product that the field constraints cannot express.": "// checkProduct aplica las reglas de un producto que las restricciones de campo no pueden expresar.",
    "// ---- batches and stock levels ----": "// ---- lotes y niveles de stock ----",
    "// levelAt is the stock level row of a batch in a location; found is false when the batch was never there.": "// levelAt es la fila de nivel de stock de un lote en una ubicación; found es falso cuando el lote nunca estuvo allí.",
    "// levelOrCreate returns the level row of a batch in a location, creating it at quantity 0 and revision 0.": "// levelOrCreate devuelve la fila de nivel de stock de un lote en una ubicación, creándola en cantidad 0 y revision 0.",
    "// applyToLevel changes the quantity of a stock level by `quantity` in direction d, guarded by the revision the caller": "// applyToLevel cambia la cantidad de un nivel de stock por `quantity` en dirección d, guardado por la revisión que el llamador",
    "// read: if the row changed since (expectedRevision differs from the stored one) it is ErrConflict and nothing is": "// leyó: si la fila cambió desde entonces (expectedRevision difiere de la almacenada) es ErrConflict y nada es",
    "// written. stock.Apply keeps the quantity from going negative (ErrInsufficientStock).": "// escrito. stock.Apply mantiene la cantidad sin ser negativa (ErrInsufficientStock).",
    "// addMovement appends one ledger entry inside the transaction.": "// addMovement añade una entrada al libro mayor dentro de la transacción.",
    "// lotsAt turns the stock of a product in a location into the lots webtyp.com/stock allocates from: one Lot per batch": "// lotsAt convierte el stock de un producto en una ubicación en los lotes que webtyp.com/stock asigna de: un Lot por lote",
    "// that has a level row there, with the batch's expiry and the level's quantity.": "// que tiene una fila de nivel allí, con la caducidad del lote y la cantidad del nivel.",
    "// ---- the four stock operations ----": "// ---- las cuatro operaciones de stock ----",
    "// ReceiveStock adds stock of a product to a location: it finds or creates the batch (the expiry date is mandatory), adds": "// ReceiveStock añade stock de un producto a una ubicación: encuentra o crea el lote (la fecha de caducidad es obligatoria), añade",
    "// the quantity to the (location, batch) level and appends a stock.Purchase movement, all in one transaction.": "// la cantidad al nivel (location, batch) y añade un movimiento stock.Purchase, todo en una transacción.",
    "// RegisterEntry adds to an EXISTING batch: an inbound correction of a count (stock.AdjustIn).": "// RegisterEntry añade a un lote EXISTENTE: una corrección entrante de un conteo (stock.AdjustIn).",
    "// RegisterOutput takes stock out: a sale, an internal use or a write-off. Without a batch id the batches are drawn by": "// RegisterOutput saca stock: una venta, uso interno o descarte. Sin un id de lote, los lotes son extraídos por",
    "// first expiry first out (stock.Allocate); only a write-off (stock.AdjustOut) may take expired stock. It returns one": "// primero en caducar primero en salir (stock.Allocate); solo un descarte (stock.AdjustOut) puede tomar stock caducado. Devuelve un",
    "// movement per batch used. If the cost fact cannot be published after the commit, the movements are returned WITH the": "// movimiento por lote usado. Si el hecho de costo no puede ser publicado después de la confirmación, los movimientos se devuelven CON el",
    "// error: the stock change is already saved.": "// error: el cambio de stock ya está guardado.",
    "// TransferStock moves stock of a product between two locations, drawing the batches by first expiry first out and": "// TransferStock mueve stock de un producto entre dos ubicaciones, extrayendo los lotes por primero en caducar primero en salir y",
    "// putting the same batches in the destination: stock.TransferOut movements at the origin, stock.TransferIn at the": "// poniendo los mismos lotes en el destino: movimientos stock.TransferOut en el origen, stock.TransferIn en el",
    "// destination, one transaction.": "// destino, una transacción.",
    "// ---- events and cost facts ----": "// ---- eventos y hechos de costo ----",
    "// publishCosts announces what an output consumed, at cost: one money.Consumed fact per movement whose batch has a unit cost.": "// publishCosts anuncia lo que un retiro consumió, al costo: un hecho money.Consumed por movimiento cuyo lote tiene costo unitario.",
    "// It does nothing unless the application gave a Publisher and a MoneyScope.": "// No hace nada a menos que la aplicación diera un Publisher y un MoneyScope.",
    "// Package seed loads demo data through the module's own methods, so every row is validated like a real one.": "// Package seed carga datos de demo usando los métodos propios del módulo, así cada fila es validada como una real.",
    "// Data is what Load created, for the demos that build on it.": "// Data es lo que Load creó, para las demos que se construyen sobre ello.",
    "// Load creates two locations (the counter is a sales point), three products and one batch of each received in the store.": "// Load crea dos ubicaciones (el mostrador es punto de venta), tres productos y un lote de cada uno recibido en la tienda.",
    "// NewProductView builds the Presenter of the products: the engine a renderer (crudview or any other) wraps. The app picks": "// NewProductView construye el Presenter de los productos: el motor que un renderizador (crudview u otro) envuelve. La app elige",
    "// the renderer; this module imports only view, model and router.": "// el renderizador; este módulo solo importa view, model y router.",
    "// NewLocationView builds the Presenter of the locations.": "// NewLocationView construye el Presenter de las ubicaciones.",
    "// Screens declares the screens this module provides to the host app's menu.": "// Screens declara las pantallas que este módulo provee al menú de la app anfitriona.",
    "// ID is the identity of the module's screen: the RBAC prefix on the server and the route of the navigation on the client.": "// ID es la identidad de la pantalla del módulo: el prefijo RBAC en el servidor y la ruta de navegación en el cliente.",
    "// Label is the text of the navigation entry.": "// Label es el texto de la entrada de navegación.",
    "// boxPath is the one drawing the three icons share: a box.": "// boxPath es el que dibuja el icono que comparten los tres: una caja.",
    "// Icons hosts IconSvg so that the app discovers the sprite of the module.": "// Icons aloja IconSvg para que la aplicación descubra el sprite del módulo.",
    "// Package migrate reconciles the schema this module owns. It is a package of its own, never called by New, so that nothing": "// Package migrate reconcilia el esquema del que este módulo es dueño. Es un paquete propio, nunca llamado por New, para que nada",
    "// on a WASM build's import path pulls webtyp.com/ddl: schema work is deploy-time work.": "// en la ruta de importación de una build WASM arrastre webtyp.com/ddl: el trabajo de esquema es trabajo de deploy-time.",
    "// Migrate creates the five tables in foreign-key order: product and location first, then the batch that references the": "// Migrate crea las cinco tablas en orden de clave externa: producto y ubicación primero, luego el lote que referencia al",
    "// product, then the stock level and the movement that reference location and batch.": "// producto, luego el nivel de stock y el movimiento que referencian ubicación y lote.",
    "// Root-level test (justified): exercises the unexported applyToLevel — the public API cannot force a stale revision.": "// Test a nivel de raíz (justificado): prueba el no exportado applyToLevel — la API pública no puede forzar una revisión obsoleta.",
    "// somebody else changes the row first: the revision moves on": "// alguien más cambia la fila primero: la revisión avanza",
    "// the second writer still holds the old revision": "// el segundo escritor todavía tiene la revisión vieja",
    "// neverExpires stands for \"does not expire\" in this module: a pharmacy batch always has an expiry date (the plan's rule),": "// neverExpires significa \"no caduca\" en este módulo: un lote de farmacia siempre tiene fecha de caducidad (regla del plan),",
    "// so the suite's lot without expiry (0) is received with a date far beyond every other, which orders it last as the": "// así que el lote de la suite sin caducidad (0) se recibe con una fecha mucho más allá de cualquier otra, lo que lo ordena último como la",
    "// suite expects; Lots() translates it back to 0.": "// suite espera; Lots() lo traduce de vuelta a 0.",
    "// testIDGen is the test double of model.IDGenerator: a module never builds its own generator.": "// testIDGen es el doble de prueba de model.IDGenerator: un módulo nunca construye su propio generador.",
    "// newModule is a Module over an in-memory database, without events.": "// newModule es un Module sobre una base de datos en memoria, sin eventos.",
    "// newProduct creates an active, taxed, for-sale product (price 1000) and fails the test if it cannot.": "// newProduct crea un producto activo, gravado, a la venta (precio 1000) y falla el test si no puede.",
    "// newLocation creates an active location and fails the test if it cannot.": "// newLocation crea una ubicación activa y falla el test si no puede.",
    "// idPair joins an id chosen by the conformance suite (\"prod1\", \"loc1\", \"lotA\") to the id the module generated.": "// idPair une un id elegido por la suite de conformidad (\"prod1\", \"loc1\", \"lotA\") al id generado por el módulo.",
    "// A slice scanned linearly, never a map: no module code or test uses map[K]V.": "// Un slice escaneado linealmente, nunca un map: ningún código o test de módulo usa map[K]V.",
    "// ledger adapts a Module to stocktest.Ledger so that the shared stock rules run against this module.": "// ledger adapta un Module a stocktest.Ledger para que las reglas de stock compartidas se ejecuten contra este módulo.",
    "// mapErr translates the module's errors to the suite's: only \"not enough stock\" differs.": "// mapErr traduce los errores del módulo a los de la suite: solo \"no enough stock\" difiere.",
    "// TestStockConformance runs the shared stock rules (first expiry first out, expired lots unused, atomic failures,": "// TestStockConformance ejecuta las reglas de stock compartidas (primero en caducar primero en salir, lotes caducados sin usar, fallas atómicas,",
    "// transfers, ledger balance, never negative) against this module.": "// transferencias, balance del libro mayor, nunca negativo) contra este módulo.",
    "// Update name of a1": "// Actualizar nombre de a1",
    "// Update a1 to have a2's sku": "// Actualizar a1 para tener el sku de a2",
    "// dummy use of newLocation just to avoid unused": "// uso ficticio de newLocation solo para evitar sin usar",
    "// mockCaller is a router.Caller that records the last call and answers it with onCall.": "// mockCaller es un router.Caller que graba la última llamada y la responde con onCall.",
    "// output of a batch with cost 0 and MoneyScope: \"x\"": "// retiro de un lote con costo 0 y MoneyScope: \"x\"",
    "// transfer and entry with cost and scope": "// transferencia y entrada con costo y alcance",
    "// guarded) and must fail the build of the app, so the module must mount all of them with a resource and an action.": "// guardado) y debe fallar la compilación de la app, así que el módulo debe montar todas ellas con un recurso y una acción.",
    "// output builds the arguments of a RegisterOutput.": "// output construye los argumentos de un RegisterOutput.",
    "// recordingSink is a money.Sink that keeps what it accepts.": "// recordingSink es un money.Sink que mantiene lo que acepta.",
    "// receive builds the arguments of a ReceiveStock for a product and a location; the tests change what they exercise.": "// receive construye los argumentos de un ReceiveStock para un producto y una ubicación; los tests cambian lo que prueban.",
    "// call runs one operation through the in-process caller, the way a transport would, and returns its error.": "// call ejecuta una operación a través del caller en proceso, de la forma en que lo haría un transporte, y devuelve su error.",
    "// the view's update: ids + the fields to change + a record with the values": "// la actualización de la vista: ids + los campos a cambiar + un registro con los valores",
    "// the view's delete: ids only; a product with batches cannot go": "// la eliminación de la vista: solo ids; un producto con lotes no puede ir",
    "// TestViewConformance runs the shared behaviour of a view.Presenter against the engine this module's views are built on.": "// TestViewConformance ejecuta el comportamiento compartido de un view.Presenter contra el motor en el que se construyen las vistas de este módulo."
}

def translate_file(path):
    with open(path) as f:
        content = f.read()
    for eng, esp in translations.items():
        content = content.replace(eng, esp)
    with open(path, 'w') as f:
        f.write(content)

for root, _, files in os.walk('.'):
    for f in files:
        if f.endswith('.go'):
            translate_file(os.path.join(root, f))

examples = {
    r"func \(m \*Module\) UpdateProduct\(": "// Ejemplo: pm.UpdateProduct(...)\nfunc (m *Module) UpdateProduct(",
    r"func \(m \*Module\) UpsertProduct\(": "// Ejemplo: pm.UpsertProduct(...)\nfunc (m *Module) UpsertProduct(",
    r"func \(m \*Module\) PatchProducts\(": "// Ejemplo: pm.PatchProducts(...)\nfunc (m *Module) PatchProducts(",
    r"func \(m \*Module\) DeactivateProduct\(": "// Ejemplo: pm.DeactivateProduct(...)\nfunc (m *Module) DeactivateProduct(",
    r"func \(m \*Module\) DeleteProducts\(": "// Ejemplo: pm.DeleteProducts(...)\nfunc (m *Module) DeleteProducts(",
    r"func \(m \*Module\) UpdateLocation\(": "// Ejemplo: pm.UpdateLocation(...)\nfunc (m *Module) UpdateLocation(",
    r"func \(m \*Module\) UpsertLocation\(": "// Ejemplo: pm.UpsertLocation(...)\nfunc (m *Module) UpsertLocation(",
    r"func \(m \*Module\) PatchLocations\(": "// Ejemplo: pm.PatchLocations(...)\nfunc (m *Module) PatchLocations(",
    r"func \(m \*Module\) DeleteLocations\(": "// Ejemplo: pm.DeleteLocations(...)\nfunc (m *Module) DeleteLocations(",
    r"func \(m \*Module\) CreateProduct\(": "// Ejemplo: pm.CreateProduct(...)\nfunc (m *Module) CreateProduct(",
    r"func \(m \*Module\) GetProduct\(": "// Ejemplo: pm.GetProduct(...)\nfunc (m *Module) GetProduct(",
    r"func \(m \*Module\) CreateLocation\(": "// Ejemplo: pm.CreateLocation(...)\nfunc (m *Module) CreateLocation(",
    r"func \(m \*Module\) GetLocation\(": "// Ejemplo: pm.GetLocation(...)\nfunc (m *Module) GetLocation(",
    r"func \(m \*Module\) ReceiveStock\(": "// Ejemplo: pm.ReceiveStock(...)\nfunc (m *Module) ReceiveStock(",
    r"func \(m \*Module\) RegisterEntry\(": "// Ejemplo: pm.RegisterEntry(...)\nfunc (m *Module) RegisterEntry(",
    r"func \(m \*Module\) RegisterOutput\(": "// Ejemplo: pm.RegisterOutput(...)\nfunc (m *Module) RegisterOutput(",
    r"func \(m \*Module\) TransferStock\(": "// Ejemplo: pm.TransferStock(...)\nfunc (m *Module) TransferStock(",
    r"func New\(": "// Ejemplo: pm, err := New(db, deps)\nfunc New(",
    r"func NewProductView\(": "// Ejemplo: view := NewProductView(caller)\nfunc NewProductView(",
    r"func NewLocationView\(": "// Ejemplo: view := NewLocationView(caller)\nfunc NewLocationView(",
    r"func Load\(": "// Ejemplo: data, err := Load(pm, tenant)\nfunc Load(",
    r"const ID ": "// Ejemplo: const id = ID\nconst ID ",
    r"const Label ": "// Ejemplo: const label = Label\nconst Label ",
    r"func Screens\(": "// Ejemplo: screens := Screens(caller)\nfunc Screens(",
    r"func Migrate\(": "// Ejemplo: err := Migrate(conn, compiler)\nfunc Migrate(",
    r"func \(m \*Icons\) IconSvg\(": "// Ejemplo: svg := icons.IconSvg()\nfunc (m *Icons) IconSvg("
}

for root, _, files in os.walk('.'):
    for f in files:
        if f.endswith('.go') and not f.endswith('_test.go'):
            path = os.path.join(root, f)
            with open(path) as fl:
                content = fl.read()
            for rgx, rep in examples.items():
                content = re.sub(rgx, rep, content)
            with open(path, 'w') as fl:
                fl.write(content)
```
   - Run `python3 translate.py`.
   - Run `git add . && git commit -m "language"`.

5. **Verify and Pre-commit Checks**
   - Verify `git diff --stat main`.
   - Verify tests run successfully using `gotest`.
   - Run `grep -rn "CreateTable(" --include=*.go . | grep -v _test.go` and ensure empty.

6. **Complete pre-commit**
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

7. **Submit**
   - Submit the PR.
