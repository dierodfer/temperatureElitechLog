package main

import (
	"flag"
	"io"
	"strings"
	"testing"
)

// El fichero debe poder escribirse antes o después de los flags: el paquete
// flag por sí solo deja de parsear en el primer argumento posicional.
func TestParseArgsAcceptsFileInAnyPosition(t *testing.T) {
	cases := map[string][]string{
		"fichero primero": {"datos.xls", "--desde", "2025-06", "--hasta", "2026-07"},
		"fichero último":  {"--desde", "2025-06", "--hasta", "2026-07", "datos.xls"},
		"fichero enmedio": {"--desde", "2025-06", "datos.xls", "--hasta", "2026-07"},
		"con igual":       {"--desde=2025-06", "datos.xls", "--hasta=2026-07"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			desde := fs.String("desde", "", "")
			hasta := fs.String("hasta", "", "")

			files, err := parseArgs(fs, args)
			if err != nil {
				t.Fatalf("parseArgs: %v", err)
			}
			if len(files) != 1 || files[0] != "datos.xls" {
				t.Errorf("posicionales = %v, se esperaba [datos.xls]", files)
			}
			if *desde != "2025-06" || *hasta != "2026-07" {
				t.Errorf("flags = %q/%q, se esperaban 2025-06/2026-07", *desde, *hasta)
			}
		})
	}
}

func TestParseArgsReportsUnknownFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if _, err := parseArgs(fs, []string{"--inventado"}); err == nil {
		t.Fatal("un flag desconocido debería dar error")
	}
}

func TestVersionString(t *testing.T) {
	if got := versionString(); !strings.Contains(got, version) {
		t.Errorf("versionString() = %q, debería contener %q", got, version)
	}
}
