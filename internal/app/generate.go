// Package app une el lector, la agregación y el informe en una sola operación,
// compartida por la línea de comandos y por el asistente interactivo.
package app

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
	"github.com/dierodfer/temperatureElitechLog/internal/report"
	"github.com/dierodfer/temperatureElitechLog/internal/stats"
)

// Request son los parámetros de una generación de informes.
type Request struct {
	Input        string
	From         string // mes inicial, AAAA-MM
	To           string // mes final, AAAA-MM
	Interval     int
	OutputDir    string
	Decimals     int
	Encoding     string
	IncludeEmpty bool
}

// Result describe lo que se ha generado, para poder informar al usuario.
type Result struct {
	Files     []string
	OutputDir string
	Blocks    int
	Skipped   int // bloques omitidos por no tener lecturas
	Readings  int
	Invalid   int // filas del Excel que no se pudieron interpretar
	Warnings  []string
}

// Generate produce un fichero TXT por bloque y devuelve el resumen.
func Generate(req Request) (*Result, error) {
	from, to, enc, err := validateRequest(req)
	if err != nil {
		return nil, err
	}

	log, err := elitech.Open(req.Input)
	if err != nil {
		return nil, err
	}

	res := &Result{
		OutputDir: req.OutputDir,
		Readings:  len(log.Readings),
		Invalid:   log.Skipped,
		Warnings:  collectWarnings(log, from, to),
	}

	days := stats.ByDay(log.Readings, log.Device.HighLimit, log.Device.LowLimit)
	blocks, err := stats.Split(days, from, to, req.Interval)
	if err != nil {
		return nil, err
	}

	opt := report.Options{
		Decimals: req.Decimals,
		From:     from,
		To:       to,
		Interval: req.Interval,
	}
	if err := writeBlocks(res, blocks, log, req, opt, enc); err != nil {
		return nil, err
	}

	if res.Blocks == 0 {
		return nil, fmt.Errorf("no hay ninguna lectura entre %s y %s: revisa el rango de meses",
			from.Human(), to.Human())
	}
	return res, nil
}

// validateRequest interpreta y valida los parámetros de la petición, antes de
// tocar el disco.
func validateRequest(req Request) (from, to stats.Month, enc report.Encoding, err error) {
	from, err = stats.ParseMonth(req.From)
	if err != nil {
		return from, to, enc, err
	}
	to, err = stats.ParseMonth(req.To)
	if err != nil {
		return from, to, enc, err
	}
	if to.Before(from) {
		return from, to, enc, fmt.Errorf("el mes inicial (%s) es posterior al final (%s)", from, to)
	}
	if req.Interval < 1 {
		return from, to, enc, fmt.Errorf("el intervalo debe ser de al menos 1 día")
	}
	if req.Decimals < 0 || req.Decimals > 6 {
		return from, to, enc, fmt.Errorf("los decimales deben estar entre 0 y 6")
	}
	enc, err = report.ParseEncoding(req.Encoding)
	return from, to, enc, err
}

// collectWarnings recopila los avisos a mostrar antes de generar los informes.
func collectWarnings(log *elitech.Log, from, to stats.Month) []string {
	var warnings []string
	if log.Skipped > 0 {
		warnings = append(warnings,
			fmt.Sprintf("%d filas del Excel no se pudieron interpretar y se han descartado", log.Skipped))
	}

	// Avisar cuando se piden meses que el registrador no llegó a cubrir evita
	// que un rango mal tecleado pase por un informe vacío sin más.
	first := stats.MonthOf(log.Readings[0].Time)
	last := stats.MonthOf(log.Readings[len(log.Readings)-1].Time)
	if from.Before(first) || last.Before(to) {
		warnings = append(warnings, fmt.Sprintf(
			"el fichero solo contiene datos de %s a %s; los meses fuera de ese margen no tendrán lecturas",
			first.Human(), last.Human()))
	}
	return warnings
}

// writeBlocks escribe el TXT de cada bloque con lecturas y actualiza res con
// lo que se ha generado.
func writeBlocks(res *Result, blocks []stats.Block, log *elitech.Log, req Request, opt report.Options, enc report.Encoding) error {
	for _, b := range blocks {
		if b.DaysWithData() == 0 && !req.IncludeEmpty {
			res.Skipped++
			continue
		}
		name := report.Filename(b, log.Device)
		if err := report.WriteFile(req.OutputDir, name, report.Render(b, log.Device, log.Source, opt), enc); err != nil {
			return fmt.Errorf("no se pudo escribir %s: %w", name, err)
		}
		res.Files = append(res.Files, name)
		res.Blocks++
	}
	return nil
}

// Print escribe un resumen legible de lo generado.
func (r *Result) Print(w io.Writer) {
	abs, err := filepath.Abs(r.OutputDir)
	if err != nil {
		abs = r.OutputDir
	}

	fmt.Fprintf(w, "\nLecturas procesadas : %d\n", r.Readings)
	fmt.Fprintf(w, "Informes generados  : %d\n", r.Blocks)
	if r.Skipped > 0 {
		fmt.Fprintf(w, "Bloques sin datos   : %d (omitidos; usa --incluir-vacios para generarlos)\n", r.Skipped)
	}
	fmt.Fprintf(w, "Carpeta de salida   : %s\n", abs)

	for _, warn := range r.Warnings {
		fmt.Fprintf(w, "\nAviso: %s\n", warn)
	}

	fmt.Fprintln(w)
	for _, f := range r.Files {
		fmt.Fprintln(w, "  "+f)
	}
}
