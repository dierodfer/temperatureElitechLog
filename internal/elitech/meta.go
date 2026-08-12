package elitech

import (
	"strings"

	"github.com/dierodfer/temperatureElitechLog/internal/xlsx"
)

// Device recoge los metadatos que ElitechLog escribe en la hoja "Resumen".
// Todos los campos son opcionales: si el fichero no los trae, el informe
// simplemente omite esa línea.
type Device struct {
	Model       string
	Serial      string
	Sensor      string
	Firmware    string
	Interval    string
	Description string
	HighLimit   *float64 // límite de alarma superior (H1)
	LowLimit    *float64 // límite de alarma inferior (L1)
}

// readDevice extrae los metadatos de la hoja de resumen. Se recorre celda a
// celda buscando etiquetas conocidas porque ElitechLog reordena los pares
// etiqueta/valor según el modelo de registrador.
func readDevice(book *xlsx.File) Device {
	sheet := summarySheet(book)
	if sheet == nil {
		return Device{}
	}

	var d Device
	for r, row := range sheet.Rows {
		for c, cell := range row {
			if text := strings.TrimSpace(cell); text != "" {
				applyCell(&d, sheet, r, c, text)
			}
		}
	}
	return d
}

// summarySheet localiza la hoja de metadatos: "Resumen", su nombre en inglés
// "Summary" o, en su defecto, la primera hoja del libro.
func summarySheet(book *xlsx.File) *xlsx.Sheet {
	if sheet := book.Sheet("Resumen"); sheet != nil {
		return sheet
	}
	if sheet := book.Sheet("Summary"); sheet != nil {
		return sheet
	}
	if len(book.Sheets) > 0 {
		return book.Sheets[0]
	}
	return nil
}

// applyCell interpreta una celda no vacía de la hoja de resumen: puede ser un
// límite de alarma embebido en el texto o la etiqueta de un campo conocido.
func applyCell(d *Device, sheet *xlsx.Sheet, r, c int, text string) {
	// Los límites de alarma vienen embebidos en el propio texto, con la
	// forma "H1: Encima:    8,0°C".
	if v, ok := limitValue(text, "h1"); ok && d.HighLimit == nil {
		d.HighLimit = &v
		return
	}
	if v, ok := limitValue(text, "l1"); ok && d.LowLimit == nil {
		d.LowLimit = &v
		return
	}

	switch normalize(text) {
	case "modelo", "model":
		assign(&d.Model, valueAfter(sheet, r, c))
	case "s/n", "sn", "numero de serie", "serial number":
		assign(&d.Serial, valueAfter(sheet, r, c))
	case "tipo de sensor", "sensor type":
		assign(&d.Sensor, valueAfter(sheet, r, c))
	case "version de firmware", "firmware version":
		assign(&d.Firmware, valueAfter(sheet, r, c))
	case "intervalo", "interval":
		assign(&d.Interval, valueAfter(sheet, r, c))
	case "descripcion", "description":
		assign(&d.Description, valueAfter(sheet, r, c))
	}
}

// assign solo escribe la primera aparición: en estos libros la etiqueta puede
// repetirse más abajo con un valor menos específico.
func assign(dst *string, v string) {
	if *dst == "" {
		*dst = v
	}
}

// valueAfter devuelve la primera celda no vacía a la derecha de la etiqueta.
func valueAfter(s *xlsx.Sheet, row, col int) string {
	for c := col + 1; c < len(s.Rows[row]); c++ {
		if v := strings.TrimSpace(s.Rows[row][c]); v != "" {
			return v
		}
	}
	return ""
}

// limitValue reconoce las líneas de configuración de alarma y devuelve su
// temperatura, p. ej. "L1: Abajo:    2,0°C" con prefijo "l1" → 2.
func limitValue(text, prefix string) (float64, bool) {
	n := normalize(text)
	if !strings.HasPrefix(n, prefix) {
		return 0, false
	}
	if idx := strings.LastIndex(text, ":"); idx >= 0 {
		text = text[idx+1:]
	}
	return parseFloat(text)
}
