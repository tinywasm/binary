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
    Null(name string); Object(name string, val Encodable); Array(name string, n int, each func(i int, a ArrayWriter))
}
type Encodable interface { EncodeFields(w FieldWriter) }
type ArrayWriter interface { String(val string); Int(val int64); Float(val float64); Bool(val bool); Bytes(val []byte); Object(val Encodable) }

type FieldReader interface {
    String(name string)(string,bool); Int(name string)(int64,bool); Uint(name string)(uint64,bool)
    Float(name string)(float64,bool); Bool(name string)(bool,bool); Bytes(name string)([]byte,bool)
    Object(name string, into Decodable) bool; Array(name string)(ArrayReader,bool)
}
type Decodable interface { DecodeFields(r FieldReader) error }
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

`Message` implementa `fmt.Encodable`/`fmt.Decodable` directamente (campos primitivos, sin reflect).
El `Payload` es el resultado de `binary.Encode(domainStruct)` donde `domainStruct` implementa
`fmt.Encodable`. La descomposición: `binary.Decode(payload, &domainStruct)` → `DecodeFields`.

## Diseño (resuelto)

### `binaryWriter` (`fmt.FieldWriter`)

Escribe al formato binario de transporte (varint + length-prefix) sobre un `io.Writer` sin `reflect`:

- `String(name, val)` → longitud (uvarint) + bytes. **El `name` se omite en binario** (el orden
  de campos es fijo por la generación de `ormc`; no se necesita el nombre en wire).
- `Int(name, val)` → zigzag + uvarint.
- `Uint(name, val)` → uvarint.
- `Float(name, val)` → 8 bytes little-endian (float64) ó 4 bytes (float32 via `Uint` + cast).
- `Bool(name, val)` → 1 byte (0/1).
- `Bytes(name, val)` → longitud (uvarint) + bytes.
- `Null(name)` → 1 byte (0 longitud / sentinel).
- `Object(name, val)` → `val.EncodeFields(w)` recursivo (no framing extra; la estructura la
  conoce el `Decodable` receptor).
- `Array(name, n, each)` → n (uvarint) + cada elemento inline.

Reutilizar `encoder.scratch [10]byte` ya presente para varint sin alloc. **Eliminar el `instance`
(singleton de caches de `reflect.Type → codec`)**: ya no es necesario.

### `binaryReader` (`fmt.FieldReader`)

Lee secuencialmente del `reader` ya existente (varint/length-prefix). Como los nombres de campo
se omiten en wire (orden fijo), cada `r.Tipo("name")` **lee el siguiente valor** en el stream
(ignora el `name` en tiempo de ejecución — solo sirve a `DecodeFields` como documentación
legible). El orden de `DecodeFields` generado por `ormc` debe coincidir con el de `EncodeFields`.

Nota: el enfoque de orden fijo (sin `name` en wire) es **la propiedad del formato binario** —
compacto y 0-alloc. El `name` existe en la interfaz para compatibilidad con el contrato
`fmt.FieldReader` que usan `json`/`jsvalue` (donde sí importa el nombre por parsing de texto).

### `ArrayReader` binario

Lee `n` (uvarint al entrar a `Array`), expone `Len() int` y lee elemento a elemento
secuencialmente. Sin `map`, sin buffer intermedio.

## Pasos de ejecución

### Stage 0 — crear `AGENTS.md`

1. Crear `/home/cesar/Dev/Project/tinywasm/binary/AGENTS.md` modelado en `fmt/AGENTS.md`:
   restricciones del ecosistema (no reflect, no map, no stdlib, agnóstico wasm+backend, gotest,
   gopush). Mencionar que `binary` es reflection-free desde esta versión vía el codec `fmt`.

### Stage 1 — `binaryWriter` y migrar `Encode`

2. Crear `codec.go` (o `writer.go`) con `binaryWriter struct` que implemente `fmt.FieldWriter`
   y `fmt.ArrayWriter`, usando el `encoder` existente como base (reusar `scratch`/`writeUvarint`/
   `writeFloat64`).
3. Cambiar la firma de `Encode`:
   ```go
   func Encode(input fmt.Encodable, output any) error
   ```
   `Encode` crea/reusa el `binaryWriter`, llama `input.EncodeFields(w)`, vuelca a `*[]byte` o
   `io.Writer`. **Eliminar el `instance` singleton** y la lógica de caché por `reflect.Type`.
4. `Message.EncodeFields(w fmt.FieldWriter)` → escribir `Topic`, `Type`, `ID`, `Payload`
   directamente con métodos tipados (sin reflect).

### Stage 2 — `binaryReader` y migrar `Decode`

5. Crear `binaryReader struct` que implemente `fmt.FieldReader` sobre el `reader` existente
   (secuencial, ignora `name`). Implementar `binaryArrayReader` para `Array`.
6. Cambiar la firma de `Decode`:
   ```go
   func Decode(input any, output fmt.Decodable) error
   ```
   (`input` sigue siendo `[]byte` o `io.Reader` — es el origen de bytes, no el dato).
7. `Message.DecodeFields(r fmt.FieldReader) error` → leer `Topic`, `Type`, `ID`, `Payload`
   en el mismo orden que `EncodeFields`.

### Stage 3 — eliminar `reflect`

8. **Eliminar** `codecs.go` (los `reflectArraycodec`, `binaryMarshalercodec`, etc. basados en
   `reflect.Value`). Eliminar `binary.go`'s `instance`/`once`/`sync`. Eliminar importaciones de
   `reflect` y `sync`.
9. Verificar que `reflect` ya no aparece en ningún `.go` del paquete (excepto `_test.go` si
   algún test lo necesitara — preferible evitarlo).

### Stage 4 — tests

10. Adaptar/reescribir los tests: los tipos de test implementan `fmt.Encodable`/`fmt.Decodable`
    a mano. Cubrir round-trip: primitivos, `[]byte`, `string`, struct anidado (`Object`), slices
    (`Array`), `Message` completo.
11. **0-alloc**: `testing.AllocsPerRun` sobre `Encode` → **0 asignaciones** (buffer reusado,
    sin reflect, sin slice intermedio).
12. `gotest` verde.

### Stage 5 — actualizar el benchmark existente — OBLIGATORIO

**YA EXISTE** `docs/BENCHMARK.md`. **NO crear** un doc nuevo; **actualizar** lo que hay:

13. **Baseline ANTES** (si el repo tiene `_bench_test.go` o equivalente): correr con la
    implementación actual basada en `reflect`; anotar `ns/op`/`B/op`/`allocs/op` y tamaño wasm.
14. **Medir DESPUÉS** (codec): re-correr. Esperado: **0 `allocs/op`** en `Encode`; sin bloque
    `reflect` (~72 KB) en binario wasm.
15. **Actualizar `docs/BENCHMARK.md`**: tabla con Antes (reflect) | Después (codec) | delta.
    Tamaño wasm antes/después. "Last updated" actualizado.

### Stage 6 — documentación — OBLIGATORIO

16. **`README.md`**: actualizar firmas de `Encode`/`Decode` (`fmt.Encodable`/`fmt.Decodable`);
    mencionar que `binary` es reflection-free. Agregar ejemplo mínimo con un struct que implementa
    `fmt.Encodable`/`fmt.Decodable`. Enlazar `docs/BENCHMARK.md`.
17. **`docs/message-envelope.md`**: verificar que el protocolo `Message` está documentado
    correctamente con la nueva firma del codec (sin `reflect`).

## Verificación (repo-local, ejecutable por el agente)

```bash
# 1. reflect eliminado del paquete (no de _test.go):
grep -rn '"reflect"' *.go | grep -v _test && echo "FALLA: reflect queda" || echo "OK"

# 2. sync eliminado (singleton instance ya no existe):
grep -rn '"sync"' *.go | grep -v _test && echo "FALLA: sync queda" || echo "OK"

# 3. sin map en el camino de serialización:
grep -nE 'map\[' *.go | grep -v _test && echo "FALLA" || echo "OK"

# 4. tests + 0-alloc:
gotest
```

## Checklist de calidad (obligatorio)

- **0-alloc** en `Encode` (medido con `AllocsPerRun`); nunca `reflect.Value`/`reflect.Type`.
- **Sin `reflect`, sin `sync`, sin `map`, sin `any`** en el camino de serialización.
  (`output any` en `Encode` es el destino `*[]byte`/`io.Writer`, no el dato).
- **Sin `instance`/singleton**: eliminado con la migración (el codec no necesita caché de tipos).
- **Orden de campos = contrato implícito del formato binario**: `EncodeFields` y `DecodeFields`
  deben escribir/leer en el mismo orden. `ormc` garantiza esto en los modelos generados.
- **`Message` implementa el codec**: `EncodeFields`/`DecodeFields` con campos tipados, sin reflect.
- Crear `AGENTS.md` si no existe.
- Reglas genéricas del ecosistema: ver `AGENTS.md`.

## Tabla de stages

| Stage | Objetivo | Entregable | Criterio de salida |
|---|---|---|---|
| 0 | AGENTS.md | `AGENTS.md` creado | restricciones ecosistema documentadas |
| 1 | Encode al codec | `binaryWriter` + `Encode(fmt.Encodable,...)` | elimina `instance`/singleton |
| 2 | Decode al codec | `binaryReader` + `Decode(..., fmt.Decodable)` | lectura secuencial, sin `name` en wire |
| 3 | Eliminar reflect | borrar `codecs.go` + imports `reflect`/`sync` | `grep reflect *.go` → vacío |
| 4 | Tests + 0-alloc | tipos test `Encodable`/`Decodable`; `AllocsPerRun==0` | `gotest` verde |
| 5 | Benchmark antes/después | actualizar `docs/BENCHMARK.md` | 0 allocs; tamaño wasm antes/después |
| 6 | Documentación | `README.md` + `docs/message-envelope.md` | firmas actualizadas; sin mención a reflect |

## Nota (coordinación)

GATEs: `fmt` (contrato) y `ormc` (genera `EncodeFields`/`DecodeFields` en los modelos reales).
`binary` puede testear con tipos propios a mano, pero los consumidores pasan modelos generados
por `ormc` → mergear DESPUÉS de `ormc`. **Nota de diseño del wire format**: los nombres de campo
se omiten en binario (formato compacto); el orden es fijo por `ormc`. Esto es compatible con el
contrato `fmt.FieldReader` (el `name` existe en la interfaz pero `binaryReader` lo ignora).
Ver `~/Dev/Project/tinywasm/docs/SIZE_OPTIMIZATION_MASTER_PLAN.md`.
