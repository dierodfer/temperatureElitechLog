// Comando elitechlog: genera informes diarios de temperatura a partir de los
// ficheros que exporta ElitechLog 8.0.
//
// Admite dos modos. Con argumentos funciona como una herramienta de línea de
// comandos normal; sin argumentos —al abrirlo con doble clic desde el
// Explorador de Windows— arranca un asistente que pregunta paso a paso.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dierodfer/temperatureElitechLog/internal/app"
)

// Inyectadas por el compilador desde el workflow de CI.
var (
	version = "dev"
	commit  = "desconocido"
	date    = "desconocida"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "\nError:", err)
		os.Exit(1)
	}
}

func run() error {
	// Sin argumentos se asume doble clic: no hay nadie mirando una consola con
	// la sintaxis de los flags a mano, así que se pregunta.
	if len(os.Args) == 1 {
		return app.Wizard(versionString())
	}

	fs := flag.NewFlagSet("elitechlog", flag.ContinueOnError)
	var (
		desde    = fs.String("desde", "", "mes inicial en formato AAAA-MM (obligatorio)")
		hasta    = fs.String("hasta", "", "mes final en formato AAAA-MM, incluido (obligatorio)")
		interval = fs.Int("intervalo", 15, "días por bloque; cada bloque genera su propio TXT")
		salida   = fs.String("salida", "informes", "carpeta donde escribir los informes")
		decimals = fs.Int("decimales", 2, "decimales de las temperaturas")
		encoding = fs.String("codificacion", "utf8bom", "codificación del TXT: utf8bom, utf8 o cp1252")
		vacios   = fs.Bool("incluir-vacios", false, "generar también los bloques sin ninguna lectura")
		showVer  = fs.Bool("version", false, "mostrar la versión y salir")
	)
	fs.Usage = func() { usage(fs) }

	files, err := parseArgs(fs, os.Args[1:])
	if err != nil {
		return err
	}

	if *showVer {
		fmt.Println(versionString())
		return nil
	}

	if len(files) != 1 {
		fs.Usage()
		return errors.New("indica exactamente un fichero de ElitechLog")
	}
	if *desde == "" || *hasta == "" {
		fs.Usage()
		return errors.New("--desde y --hasta son obligatorios")
	}

	res, err := app.Generate(app.Request{
		Input:        files[0],
		From:         *desde,
		To:           *hasta,
		Interval:     *interval,
		OutputDir:    *salida,
		Decimals:     *decimals,
		Encoding:     *encoding,
		IncludeEmpty: *vacios,
	})
	if err != nil {
		return err
	}

	res.Print(os.Stdout)
	return nil
}

// parseArgs permite escribir el fichero antes o después de los flags. El
// paquete flag deja de parsear en cuanto encuentra un argumento posicional, y
// la forma natural de teclear esto es «elitechlog datos.xls --desde ...».
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func usage(fs *flag.FlagSet) {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `elitechlog %s

Calcula la media, el mínimo y el máximo de cada día a partir de un fichero
exportado por ElitechLog, y escribe un informe TXT por cada bloque de días.

Uso:
  %s <fichero.xls> --desde AAAA-MM --hasta AAAA-MM [opciones]
  %s                (sin argumentos arranca el asistente interactivo)

Ejemplo:
  %s datos.xls --desde 2025-06 --hasta 2026-07 --intervalo 15

Opciones:
`, versionString(), exe, exe, exe)
	fs.PrintDefaults()
}

func versionString() string {
	return fmt.Sprintf("%s (commit %s, %s)", version, commit, date)
}
