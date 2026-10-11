// Package report escribe los informes de texto, un fichero por bloque.
package report

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
	"github.com/dierodfer/temperatureElitechLog/internal/stats"
)

const (
	width    = 78
	dateFmt  = "02-01-2006"
	stampFmt = "02-01-2006 15:04:05"
)

// Options controla el formato de la salida.
type Options struct {
	Decimals int         // decimales de las temperaturas
	From     stats.Month // mes inicial solicitado, solo informativo
	To       stats.Month // mes final solicitado, solo informativo
	Interval int
}

// Render compone el texto del informe de un bloque: la tabla de lecturas y, al
// final, el resumen mensual y los datos del origen en un par de líneas.
func Render(b stats.Block, dev elitech.Device, source string, opt Options) string {
	var sb strings.Builder
	rule := strings.Repeat("=", width)
	thin := strings.Repeat("-", width)

	sb.WriteString("  " + pad("Fecha y hora", 21) + padLeft("Temp. °C", 12) +
		strings.Repeat(" ", 10) + "Resumen del día\n")
	sb.WriteString(thin + "\n")

	for _, d := range b.Days {
		writeDay(&sb, d, opt.Decimals)
	}

	s := b.Summarize(dev.HighLimit, dev.LowLimit)
	sb.WriteString(thin + "\n")
	if s.Count == 0 {
		sb.WriteString("  Resumen mensual: sin lecturas en este intervalo.\n")
	} else {
		sb.WriteString(fmt.Sprintf("  Resumen mensual: Media %s °C · Mínimo %s °C · Máximo %s °C\n",
			num(s.Mean, opt.Decimals), num(s.Min, opt.Decimals), num(s.Max, opt.Decimals)))
	}
	sb.WriteString("  " + footer(b, dev, source, opt) + "\n")
	sb.WriteString(rule + "\n")
	return sb.String()
}

// footer junta en una línea los datos del origen que no vienen en la tabla.
func footer(b stats.Block, dev elitech.Device, source string, opt Options) string {
	parts := []string{}
	for _, p := range []string{deviceLabel(dev), filepath.Base(source)} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	parts = append(parts,
		fmt.Sprintf("%s a %s", b.From.Format(dateFmt), b.To.Format(dateFmt)))
	return strings.Join(parts, " · ")
}

// Filename construye el nombre del fichero de un bloque.
func Filename(b stats.Block, dev elitech.Device) string {
	parts := []string{"informe"}
	if sn := sanitize(dev.Serial); sn != "" {
		parts = append(parts, sn)
	}
	parts = append(parts, b.From.Format("2006-01-02"), b.To.Format("2006-01-02"))
	return strings.Join(parts, "_") + ".txt"
}

func deviceLabel(dev elitech.Device) string {
	switch {
	case dev.Model != "" && dev.Serial != "":
		return fmt.Sprintf("%s (S/N %s)", dev.Model, dev.Serial)
	case dev.Model != "":
		return dev.Model
	case dev.Serial != "":
		return "S/N " + dev.Serial
	}
	return ""
}

// writeDay escribe cada lectura del día en su propia línea y, a la derecha,
// el máximo, el mínimo y la media del día, empezando en la segunda lectura.
// Si el día tiene pocas lecturas se añaden líneas para que el resumen quepa.
func writeDay(sb *strings.Builder, d stats.Day, decimals int) {
	if d.Count == 0 {
		sb.WriteString("  " + pad(d.Date.Format(dateFmt), 21) + padLeft("sin datos", 12) + "\n")
		return
	}

	summary := []string{
		padLeft(num(d.Max, decimals), 10) + " Máximo",
		padLeft(num(d.Min, decimals), 10) + " Mínimo",
		padLeft(num(d.Mean, decimals), 10) + " Media",
	}
	lines := max(len(d.Readings), len(summary)+1)

	for i := 0; i < lines; i++ {
		left := strings.Repeat(" ", 33)
		if i < len(d.Readings) {
			r := d.Readings[i]
			left = pad(r.Time.Format(stampFmt), 21) + padLeft(num(r.Temp, decimals), 12)
		}
		line := "  " + left
		if j := i - 1; j >= 0 && j < len(summary) {
			line += "     " + summary[j]
		}
		sb.WriteString(strings.TrimRight(line, " ") + "\n")
	}
}

// writeField omite las líneas cuyo valor no venía en el fichero de origen.

// num formatea con coma decimal, que es la convención del fichero de origen.
func num(v float64, decimals int) string {
	s := fmt.Sprintf("%.*f", decimals, v)
	// %.*f puede devolver "-0,00" al redondear negativos muy pequeños.
	if strings.Trim(s, "-0.") == "" {
		s = strings.TrimPrefix(s, "-")
	}
	return strings.ReplaceAll(s, ".", ",")
}

// pad y padLeft cuentan runas, no bytes: los acentos ocuparían dos posiciones
// con el relleno de fmt y descuadrarían las columnas.
func pad(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func padLeft(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return strings.Repeat(" ", w-n) + s
	}
	return s
}

// sanitize deja un texto utilizable como parte de un nombre de fichero.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}
