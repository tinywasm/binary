# PLAN — `binary` al codec tipado: `Encode`/`Decode` 0-alloc (eliminar `reflect`) · BREAKING

> Este plan se despacha vía el workflow CodeJob. Ver skill: `agents-workflow`.
> **Estado:** LISTO PARA REVISIÓN DEL USUARIO.
> **Repo objetivo:** `github.com/tinywasm/binary`.
> **Depende de (GATE):** `tinywasm/fmt` con el contrato del codec publicado (`fmt/docs/PLAN.md`)
> y `ormc` generando `EncodeFields`/`DecodeFields` en los modelos (`orm/docs/PLAN.md`).
> **Tipo:** breaking change (firmas de `Encode`/`Decode`: `any` → `fmt.Encodable`/`fmt.Decodable`).
> **Objetivo:** eliminar `reflect` de `binary` y migrar al contrato tipado (`fmt.FieldWriter`/
> `FieldReader`) — misma forma que `json` y `jsvalue`. 0-alloc, map-free, reflection-free.

## Reglas permanentes del repo → `AGENTS.md`

Las restricciones del ecosistema (no stdlib → `tinywasm/fmt`; **no `map`**; **no `reflect`**;
agnóstico compila wasm+backend; 0-alloc; `gotest` no `go test`) deben estar en
[`AGENTS.md`](../AGENTS.md). Este plan NO las repite completas; solo inlinea lo crítico de la
tarea (ver Checklist). **Crear `AGENTS.md` si no existe** (ver modelo en `fmt/AGENTS.md`).

## Prerequisito (PRIMERO — entorno del agente)

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Usar `gotest` (sin argumentos); **NO** `go test` directo.

## Contexto y motivación (autocontenido)

`binary` HOY serializa con `reflect`: el `instance` cachea codecs por tipo Go
(`reflect.Type → codec`), y cada `codec` opera sobre `reflect.Value`. Esto arrastra `reflect`
(~72 KB en TinyGo) + `sync` (pool/singleton) hacia cualquier binario que importe `binary`.

El codec tipado de `fmt` elimina `reflect` del camino de serialización: el modelo implementa
`fmt.Encodable`/`fmt.Decodable` y llama métodos tipados directamente. `binary` pasa a ser un
**formato de transporte** (varint/length-prefix) sobre el mismo contrato que `json`/`jsvalue`.

### Contrato de `fmt` (ya publicado, referencia)

```go
type FieldWriter interface {
    String(name, val string); Int(name string, val int64); Uint(name string, val uint64)
    Float(name string, val float64); Bool(name string, val bool); Bytes(name string, val []byte)
    Null(name string); Object(name string, val Encodable); Array(name string, n int) ArrayWriter
}
type ArrayWriter interface { String(val string); Int(val int64); Float(val float64); Bool(val bool); Bytes(val []byte); Object(val Encodable) }
type Encodable interface { EncodeFields(w FieldWriter); IsNil() bool }
type FieldReader interface {
    String(name string)(string,bool); Int(name string)(int64,bool); Uint(name string)(uint64,bool)
    Float(name string)(float64,bool); Bool(name string)(bool,bool); Bytes(name string)([]byte,bool)
    Object(name string, into Decodable) bool; Array(name string)(ArrayReader,bool)
}
type ArrayReader interface { Len() int; String(i int) string; Int(i int) int64; Float(i int) float64; Bool(i int) bool; Bytes(i int) []byte; Object(i int,into Decodable) bool }
type Decodable interface { DecodeFields(r FieldReader) error; IsNil() bool }
```

### `Message` (sobre el codec)

`Message` (`message.go`) es el envelope estándar inter-módulo:

```go
type Message struct {
    Topic   string
    Type    fmt.MessageType
    ID      uint32
    Payload []byte  // cuerpo ya serializado por binary.Encode(domainStruct)
}
```

`Message` implementa `fmt.Encodable`/`fmt.Decodable` directamente (campos primitivos, sin reflect, e `IsNil() bool { return m == nil }`).
El `Payload` es el resultado de `binary.Encode(domainStruct)` donde `domainStruct` implementa
`fmt.Encodable`. La descomposición: `binary.Decode(payload, &domainStruct)` → `DecodeFields`.

## Diseño

### `binaryWriter` (`fmt.FieldWriter` & `fmt.ArrayWriter`)

Escribe al formato binario de transporte (varint + length-prefix) sobre un `io.Writer` sin `reflect`:

- `String(name, val)` → longitud (uvarint) + bytes. **El `name` se omite en binario** (el orden
  de campos es fijo por la generación de `ormc`; no se necesita el nombre en wire).
- `Int(name, val)` → zigzag + uvarint.
- `Uint(name, val)` → uvarint.
- `Float(name, val)` → 8 bytes little-endian (float64) ó 4 bytes (float32 via `Uint` + cast).
- `Bool(name, val)` → 1 byte (0/1).
- `Bytes(name, val)` → longitud (uvarint) + bytes.
- `Null(name)` → 1 byte sentinel (ej. 0 longitud / especial).
- `Object(name, val)` → si es nil (`fmt.IsNil(val)`) escribe null-sentinel, si no `val.EncodeFields(w)` recursivo.
- `Array(name, n)` → n (uvarint) + retorna `binaryArrayWriter` (pre-alocado o estructurado) para escribir cada elemento inline.

Reutilizar `encoder.scratch [10]byte` ya presente para varint sin alloc. **Eliminar el `instance`
(singleton de caches de `reflect.Type → codec`)**: ya no es necesario.

### `binaryReader` (`fmt.FieldReader` & `fmt.ArrayReader`)

Lee secuencialmente del `reader` ya existente (varint/length-prefix). Como los nombres de campo
se omiten en wire (orden fijo), cada `r.Tipo("name")` **lee el siguiente valor** en el stream
(ignora el `name` en tiempo de ejecución). El orden de `DecodeFields` generado por `ormc` debe coincidir con el de `EncodeFields`.

- `Array(name)` → lee `n` (uvarint) + retorna `binaryArrayReader`.

## Pasos de ejecución

### Stage 0 — crear `AGENTS.md`
1. Crear `binary/AGENTS.md` modelado en `fmt/AGENTS.md`.

### Stage 1 — `binaryWriter` y migrar `Encode`
2. Crear `codec.go` con `binaryWriter struct` que implemente `fmt.FieldWriter` y `fmt.ArrayWriter`.
3. Cambiar la firma de `Encode`:
   ```go
   func Encode(input fmt.Encodable, output any) error
   ```
   Validar `fmt.IsNil(input)` escribiendo un sentinel de valor nulo. Llamar `input.EncodeFields(w)`.
   **Eliminar el `instance` singleton** y la lógica de caché por `reflect.Type`.

### Stage 2 — `binaryReader` y migrar `Decode`
4. Crear `binaryReader struct` que implemente `fmt.FieldReader` y `fmt.ArrayReader`.
5. Cambiar la firma de `Decode`:
   ```go
   func Decode(input any, output fmt.Decodable) error
   ```
   Validar `fmt.IsNil(output)` retornando error de destino nil.

### Stage 3 — eliminar `reflect`
6. **Eliminar** `codecs.go`. Eliminar `binary.go`'s `instance`/`once`/`sync`. Eliminar importaciones de `reflect` y `sync`.
7. Verificar que `reflect` ya no aparece en ningún `.go` del paquete (excepto tests).

### Stage 4 — tests
8. Adaptar/reescribir los tests: los tipos de test implementan `fmt.Encodable`/`fmt.Decodable` e `IsNil() bool { return m == nil }`. Cubrir round-trip: primitivos, `[]byte`, `string`, struct anidado, slices, typed nil pointer, `Message` completo.
9. **0-alloc**: `testing.AllocsPerRun` sobre `Encode` → **0 asignaciones**.
10. `gotest` verde.

### Stage 5 — actualizar el benchmark existente — OBLIGATORIO
11. Correr benchmarks de `docs/BENCHMARK.md` antes y después de la migración.
12. Registrar delta de `allocs/op` (esperado: 0 allocs en `Encode`) y tamaño WASM (reducción por quitar `reflect`).

### Stage 6 — documentación — OBLIGATORIO
13. Actualizar `README.md` y `docs/message-envelope.md` documentando las nuevas firmas e `IsNil()`. Enlazar a `docs/BENCHMARK.md`.
