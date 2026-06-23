# PLAN — Implementar `Raw()` en `binaryWriter` y `binaryReader`

> **Repo:** `github.com/tinywasm/binary`
> **Archivo:** `codec.go`
> **Tipo:** implementación de nueva interfaz
> **Prerequisito:** `tinywasm/fmt` publicado con `Raw()` en `FieldWriter`/`FieldReader`

## Contexto

`tinywasm/fmt` extendió `FieldWriter` con `Raw(name, val string)` y `FieldReader`
con `Raw(name string) (string, bool)`. El paquete `binary` implementa ambas
interfaces con `binaryWriter` y `binaryReader` en `codec.go`, y debe satisfacer
los nuevos métodos.

El protocolo binario no usa JSON inline — `Raw` puede implementarse como alias de
`String` sin pérdida de semántica para los consumidores actuales.

## Cambios en `codec.go`

### `binaryWriter` — agregar `Raw`

```go
func (w *binaryWriter) Raw(name, val string) {
    w.String(name, val)
}
```

### `binaryReader` — agregar `Raw`

```go
func (br *binaryReader) Raw(name string) (string, bool) {
    return br.String(name)
}
```

### `binaryArrayWriter` — agregar `Close()`

`ArrayWriter` interface ahora requiere `Close()`. Agregar stub:

```go
func (w *binaryArrayWriter) Close() {
    // protocolo binario no necesita delimitador de cierre
}
```

## Actualizar dependencia

```bash
go get github.com/tinywasm/fmt@latest
go mod tidy
```

## Verificación

```bash
go vet ./...
gotest
```

## Checklist

- [ ] `go get github.com/tinywasm/fmt@latest` actualizado
- [ ] `binaryWriter.Raw()` implementado en `codec.go`
- [ ] `binaryReader.Raw()` implementado en `codec.go`
- [ ] `binaryArrayWriter.Close()` implementado en `codec.go`
- [ ] `go vet ./...` sin errores
- [ ] `gotest` verde
