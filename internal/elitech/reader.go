// Package elitech interpreta los libros exportados por ElitechLog 8.0: localiza
// la hoja de lecturas, deduce el formato de fecha y decimal, y extrae los
// metadatos del dispositivo.
package elitech

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dierodfer/temperatureElitechLog/internal/xlsx"
)

// Reading es una lectura individual del registrador.
type Reading struct {
	Time time.Time
	Temp float64
}

// Log es el contenido útil de un fichero de ElitechLog.
type Log struct {
	Source     string    // nombre del fichero de origen
	Device     Device    // metadatos de la hoja "Resumen"
	Readings   []Reading // ordenadas cronológicamente
	Skipped    int       // filas descartadas por fecha o temperatura ilegible
	TimeLayout string    // layout detectado, útil para diagnosticar
}

// dateLayouts son los formatos de fecha que ElitechLog emite según el idioma y
// la configuración regional del equipo donde se exportó. Se prueban en orden y
// gana el primero capaz de interpretar todas las filas, lo que resuelve la
// ambigüedad entre día y mes sin tener que adivinar.
var dateLayouts = []string{
	"02-01-2006 15:04:05",
	"02/01/2006 15:04:05",
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	"01-02-2006 15:04:05",
	"01/02/2006 15:04:05",
	"02-01-2006 15:04",
	"02/01/2006 15:04",
	"2006-01-02 15:04",
	"01/02/2006 15:04",
	"02-01-2006 03:04:05 PM",
	"01/02/2006 03:04:05 PM",
}

// Open lee un fichero de ElitechLog completo.
func Open(filename string) (*Log, error) {
	book, err := xlsx.Open(filename)
	if err != nil {
		return nil, err
	}
	return parse(book, filename)
}

func parse(book *xlsx.File, filename string) (*Log, error) {
	sheet, timeCol, tempCol, headerRow, err := findDataSheet(book)
	if err != nil {
		return nil, err
	}

	body := sheet.Rows[headerRow+1:]

	raw := make([][2]string, 0, len(body))
	for i := range body {
		ts := strings.TrimSpace(sheet.Cell(headerRow+1+i, timeCol))
		tv := strings.TrimSpace(sheet.Cell(headerRow+1+i, tempCol))
		if ts == "" && tv == "" {
			continue
		}
		raw = append(raw, [2]string{ts, tv})
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("la hoja %q no contiene ninguna lectura", sheet.Name)
	}

	layout := detectLayout(raw)

	log := &Log{Source: filename, TimeLayout: layout}
	for _, r := range raw {
		ts, ok := parseTime(r[0], layout)
		if !ok {
			log.Skipped++
			continue
		}
		temp, ok := parseFloat(r[1])
		if !ok {
			log.Skipped++
			continue
		}
		log.Readings = append(log.Readings, Reading{Time: ts, Temp: temp})
	}
	if len(log.Readings) == 0 {
		return nil, fmt.Errorf("no se pudo interpretar ninguna lectura de la hoja %q: "+
			"revisa el formato de fecha (%d filas descartadas)", sheet.Name, log.Skipped)
	}

	sort.Slice(log.Readings, func(i, j int) bool {
		return log.Readings[i].Time.Before(log.Readings[j].Time)
	})

	log.Device = readDevice(book)
	return log, nil
}

// findDataSheet localiza la hoja de lecturas y las columnas de tiempo y
// temperatura. Prefiere la hoja "Lista" pero acepta cualquiera cuya cabecera
// contenga ambas columnas, para tolerar exportaciones en otros idiomas.
func findDataSheet(book *xlsx.File) (sheet *xlsx.Sheet, timeCol, tempCol, headerRow int, err error) {
	candidates := make([]*xlsx.Sheet, 0, len(book.Sheets))
	if s := book.Sheet("Lista"); s != nil {
		candidates = append(candidates, s)
	}
	for _, s := range book.Sheets {
		if len(candidates) == 0 || s != candidates[0] {
			candidates = append(candidates, s)
		}
	}

	for _, s := range candidates {
		if row, tc, pc, ok := findHeader(s); ok {
			return s, tc, pc, row, nil
		}
	}
	return nil, 0, 0, 0, fmt.Errorf("no se encontró ninguna hoja con columnas de tiempo y " +
		"temperatura; ¿es un fichero exportado por ElitechLog?")
}

// findHeader busca en las primeras filas una que declare a la vez una columna
// de tiempo y otra de temperatura.
func findHeader(s *xlsx.Sheet) (row, timeCol, tempCol int, ok bool) {
	limit := min(len(s.Rows), 30)
	for r := 0; r < limit; r++ {
		timeCol, tempCol = -1, -1
		for c, cell := range s.Rows[r] {
			switch {
			case timeCol < 0 && isTimeHeader(cell):
				timeCol = c
			case tempCol < 0 && isTempHeader(cell):
				tempCol = c
			}
		}
		if timeCol >= 0 && tempCol >= 0 {
			return r, timeCol, tempCol, true
		}
	}
	return 0, 0, 0, false
}

func isTimeHeader(cell string) bool {
	switch normalize(cell) {
	case "tiempo", "hora", "fecha", "fecha/hora", "fecha y hora", "time", "date", "date/time", "datetime", "heure":
		return true
	}
	return false
}

func isTempHeader(cell string) bool {
	n := normalize(cell)
	// La cabecera real trae la unidad pegada: "Temperatura°C".
	return strings.HasPrefix(n, "temperatura") || strings.HasPrefix(n, "temperature") || n == "temp"
}

// normalize deja el texto en minúsculas y sin acentos ni símbolos de unidad,
// para poder comparar cabeceras entre idiomas y exportaciones.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case 'á', 'à', 'ä', 'â', 'ã':
			b.WriteRune('a')
		case 'é', 'è', 'ë', 'ê':
			b.WriteRune('e')
		case 'í', 'ì', 'ï', 'î':
			b.WriteRune('i')
		case 'ó', 'ò', 'ö', 'ô', 'õ':
			b.WriteRune('o')
		case 'ú', 'ù', 'ü', 'û':
			b.WriteRune('u')
		case 'ñ':
			b.WriteRune('n')
		case 'ç':
			b.WriteRune('c')
		case '°', 'º', ' ':
			// se descarta
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '/' || r == '.' {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// detectLayout elige el primer layout capaz de parsear todas las filas. Si
// ninguno lo consigue (hay filas corruptas), se queda con el que más acierte,
// dejando que esas filas caigan en el contador de descartadas.
func detectLayout(raw [][2]string) string {
	best, bestHits := dateLayouts[0], -1
	for _, layout := range dateLayouts {
		hits := 0
		for _, r := range raw {
			if _, ok := time.Parse(layout, r[0]); ok == nil {
				hits++
			}
		}
		if hits == len(raw) {
			return layout
		}
		if hits > bestHits {
			best, bestHits = layout, hits
		}
	}
	return best
}

// parseTime interpreta la celda de fecha, admitiendo también el serial numérico
// de Excel por si alguna exportación emite la columna con formato de fecha.
func parseTime(s, layout string) (time.Time, bool) {
	if t, err := time.Parse(layout, s); err == nil {
		return t, true
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	if serial, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64); err == nil && serial > 0 {
		return excelSerialToTime(serial), true
	}
	return time.Time{}, false
}

// excelSerialToTime convierte un serial de Excel a fecha. La época es
// 1899-12-30 por el bug histórico del año bisiesto 1900 que Excel conserva.
func excelSerialToTime(serial float64) time.Time {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := math.Floor(serial)
	secs := math.Round((serial - days) * 86400)
	return epoch.AddDate(0, 0, int(days)).Add(time.Duration(secs) * time.Second)
}

// parseFloat acepta indistintamente coma o punto como separador decimal, y
// descarta la unidad si viniera pegada al número.
func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r == '-', r == '+', r == '.', r == ',':
			return r
		default:
			return -1
		}
	}, s)
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
