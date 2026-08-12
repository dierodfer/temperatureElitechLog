# temperatureElitechLog

[![build](https://img.shields.io/github/actions/workflow/status/dierodfer/temperatureElitechLog/build.yml?branch=main&label=build)](https://github.com/dierodfer/temperatureElitechLog/actions/workflows/build.yml)
[![versión](https://img.shields.io/github/v/release/dierodfer/temperatureElitechLog?label=versi%C3%B3n&color=blue)](https://github.com/dierodfer/temperatureElitechLog/releases/latest)
[![descargas](https://img.shields.io/github/downloads/dierodfer/temperatureElitechLog/total?label=descargas)](https://github.com/dierodfer/temperatureElitechLog/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/dierodfer/temperatureElitechLog?label=go)](go.mod)
[![plataformas](https://img.shields.io/badge/plataformas-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/dierodfer/temperatureElitechLog/releases/latest)
[![dependencias](https://img.shields.io/badge/dependencias-ninguna-brightgreen)](go.mod)

Repositorio dedicado a ofrecer una solución cómoda para archivos generados por ElitechLog 8.0 y su
tratamiento profesional.

`elitechlog` toma el fichero que exporta el software del registrador y calcula la **media, el mínimo
y el máximo de cada día**, dentro de un **rango de meses**, escribiendo un **informe TXT por cada
bloque de días** (quincenas por defecto).

Es un único ejecutable sin dependencias: no hace falta instalar Python, .NET ni ningún runtime.

---

## Descarga

Descarga el ejecutable de la última versión desde la página de
[Releases](../../releases/latest):

| Fichero | Sistema |
|---|---|
| `elitechlog_vX.Y.Z_windows_amd64.exe` | Windows 64 bits |
| `elitechlog_vX.Y.Z_linux_amd64` | Linux 64 bits |
| `elitechlog_vX.Y.Z_darwin_arm64` | macOS con Apple Silicon (M1 o posterior) |
| `elitechlog_vX.Y.Z_darwin_amd64` | macOS con Intel |

En Linux y macOS, dale permiso de ejecución antes de lanzarlo:

```bash
chmod +x elitechlog_vX.Y.Z_darwin_arm64
./elitechlog_vX.Y.Z_darwin_arm64
```

> **Aviso de SmartScreen / Gatekeeper.** El binario no está firmado con un certificado de código,
> así que la primera vez el sistema avisará. En **Windows**, *"Windows protegió su PC"* → **Más
> información → Ejecutar de todas formas**. En **macOS**, *"no se puede abrir porque su desarrollador
> no se puede verificar"* → Ajustes → Privacidad y seguridad → **Abrir de todas formas** (o clic
> derecho sobre el fichero → Abrir). Puedes verificar que la descarga es la publicada comparándola
> con `SHA256SUMS.txt`:
>
> ```powershell
> # Windows (PowerShell)
> Get-FileHash .\elitechlog_v1.0.0_windows_amd64.exe -Algorithm SHA256
> ```
>
> ```bash
> # Linux / macOS
> shasum -a 256 elitechlog_v1.0.0_darwin_arm64
> ```

## Uso

### Con doble clic (asistente)

Deja el `.exe` en la misma carpeta que el fichero exportado y ábrelo. El asistente pregunta paso a
paso y propone valores por defecto (basta con pulsar Intro):

```
===============================================
  Informes de temperatura - ElitechLog
===============================================

Fichero exportado por ElitechLog [ddf0774c-001.xls]:
Mes inicial (AAAA-MM) [2021-01]: 2025-06
Mes final   (AAAA-MM) [2026-06]: 2026-07
Días por informe [15]:
Carpeta de salida [informes]:
```

### Desde `cmd` o PowerShell

```
elitechlog.exe datos.xls --desde 2025-06 --hasta 2026-07 --intervalo 15
```

| Opción | Por defecto | Descripción |
|---|---|---|
| `--desde AAAA-MM` | *obligatorio* | Primer mes del rango, incluido |
| `--hasta AAAA-MM` | *obligatorio* | Último mes del rango, incluido |
| `--intervalo N` | `15` | Días por informe |
| `--salida CARPETA` | `informes` | Dónde escribir los TXT |
| `--decimales N` | `2` | Decimales de las temperaturas |
| `--codificacion` | `utf8bom` | `utf8bom`, `utf8` o `cp1252` |
| `--incluir-vacios` | *desactivado* | Generar también los bloques sin lecturas |
| `--version` | | Mostrar la versión y salir |

## Cómo se reparten los días

Los bloques se cortan **por meses naturales**: se cuenta desde el día 1 y ningún bloque cruza un
cambio de mes. Con el valor por defecto de 15 días salen las quincenas **1-15** y **16-fin de mes**:

| Mes | Bloques con `--intervalo 15` |
|---|---|
| Junio (30 días) | 01-15, 16-30 |
| Julio (31 días) | 01-15, 16-31 |
| Febrero (28 días) | 01-15, 16-28 |

El sobrante final se absorbe en el bloque anterior cuando no llega a medio intervalo, que es lo que
evita que un mes de 31 días acabe con un bloque suelto de un solo día. Con `--intervalo 10` en un
mes de 31 días salen 01-10, 11-20 y 21-31.

Los bloques sin ninguna lectura no generan fichero, salvo que se pida `--incluir-vacios`.

## Cómo es el informe

Cada bloque produce un fichero `informe_<S/N>_<desde>_<hasta>.txt`:

```
==============================================================================
  INFORME DE TEMPERATURA - ElitechLog
==============================================================================
  Dispositivo           : RC-5  (S/N 001)
  Sensor                : Temperatura
  Intervalo de registro : 6h
  Fichero de origen     : ddf0774c-001.xls
  Rango solicitado      : junio 2025 a julio 2026
  Bloque                : 16-06-2026 a 30-06-2026  (15 días)
  Límites de alarma     : L1 2,00 °C  /  H1 8,00 °C
  Generado              : 09-08-2026 22:36:35
------------------------------------------------------------------------------
  Fecha        Lecturas      Media     Mínimo     Máximo   Alarma
------------------------------------------------------------------------------
  16-06-2026          4       2,30       0,90       3,40        1
  17-06-2026          4       2,25       0,10       4,20        2
  ...
  29-06-2026          -                        sin datos
------------------------------------------------------------------------------
  RESUMEN DEL BLOQUE
    Días con datos  : 13 de 15
    Lecturas        : 51
    Media           : 3,58 °C
    Mínimo          : 0,10 °C  (17-06-2026 06:39:33)
    Máximo          : 8,10 °C  (23-06-2026 12:39:33)
    Fuera de rango  : 11 lecturas  (1 por encima de 8,00 °C / 10 por debajo de 2,00 °C)
==============================================================================
```

La columna **Alarma** y la línea **Fuera de rango** solo aparecen si el fichero trae configurados los
límites `H1`/`L1` del registrador. Los TXT se escriben con saltos de línea CRLF y, por defecto, con
BOM UTF-8, para que el Bloc de notas y Excel muestren bien los acentos y el símbolo de grado.

## Sobre el fichero de entrada

ElitechLog 8.0 exporta con extensión `.xls` un fichero que en realidad es un **XLSX** (OOXML). La
herramienta lo detecta por su contenido, no por la extensión, así que funciona con `.xls` y `.xlsx`
indistintamente.

Se interpretan automáticamente:

- La hoja de datos (`Lista`, o cualquiera con columnas de tiempo y temperatura, también en inglés).
- El formato de fecha, probando varios y quedándose con el que encaje en todas las filas: así no hay
  ambigüedad entre día-mes y mes-día.
- El separador decimal, coma o punto.
- Los metadatos de la hoja `Resumen`: modelo, S/N, sensor, intervalo y límites de alarma.

Si el fichero es un `.xls` binario antiguo o un HTML renombrado, la herramienta lo dice y pide que se
reexporte en formato Excel.

## Desarrollo

```bash
go test ./... -race   # tests
make windows           # compila dist/elitechlog.exe desde cualquier sistema
make darwin            # compila dist/elitechlog_darwin_{amd64,arm64}
make build             # Windows, Linux y macOS
```

El proyecto usa **solo la librería estándar de Go**: el lector de XLSX está implementado con
`archive/zip` y `encoding/xml` en `internal/xlsx`.

| Paquete | Responsabilidad |
|---|---|
| `internal/xlsx` | Lector OOXML mínimo |
| `internal/elitech` | Interpreta el formato de ElitechLog y sus metadatos |
| `internal/stats` | Agregación diaria y troceado en bloques |
| `internal/report` | Renderizado y escritura de los TXT |
| `internal/app` | Orquestación compartida por CLI y asistente |
| `cmd/elitechlog` | Interfaz de línea de comandos y asistente |

### Publicación

Cada push a `main` lanza los tests y, si pasan, publica automáticamente una versión nueva subiendo el
**minor** (`v1.0.0` → `v1.1.0` → `v1.2.0`), con su tag y su Release con los binarios y el
`SHA256SUMS.txt`. Para subir de **major** basta con empujar el tag a mano (`v2.0.0`): la numeración
automática continúa desde ahí. Un commit con `[skip release]` en el mensaje ejecuta los tests pero no
publica.
