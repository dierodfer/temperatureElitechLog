package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
	"github.com/dierodfer/temperatureElitechLog/internal/stats"
)

func sample(t *testing.T) (stats.Block, elitech.Device, Options) {
	t.Helper()

	high, low := 8.0, 2.0
	dev := elitech.Device{
		Model: "RC-5", Serial: "001", Sensor: "Temperatura", Interval: "6h",
		HighLimit: &high, LowLimit: &low,
	}

	readings := []elitech.Reading{
		{Time: time.Date(2025, 6, 1, 0, 39, 33, 0, time.UTC), Temp: 1.5},
		{Time: time.Date(2025, 6, 1, 12, 39, 33, 0, time.UTC), Temp: 5.5},
		{Time: time.Date(2025, 6, 2, 12, 39, 33, 0, time.UTC), Temp: 9.0},
	}

	june := stats.Month{Year: 2025, Month: time.June}
	blocks, err := stats.Split(stats.ByDay(readings, &high, &low), june, june, 15)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	opt := Options{
		Decimals:  2,
		From:      june,
		To:        june,
		Interval:  15,
		Generated: time.Date(2026, 8, 9, 16, 29, 44, 0, time.UTC),
	}
	return blocks[0], dev, opt
}

func TestRenderGolden(t *testing.T) {
	block, dev, opt := sample(t)
	got := Render(block, dev, filepath.Join("datos", "ddf0774c-001.xls"), opt)

	golden := filepath.Join("testdata", "bloque.txt")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("leyendo el fichero de referencia (regenéralo con UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Errorf("el informe no coincide con testdata/bloque.txt.\n--- obtenido ---\n%s", got)
	}
}

func TestRenderOmitsAlarmColumnWithoutLimits(t *testing.T) {
	block, _, opt := sample(t)
	out := Render(block, elitech.Device{Model: "RC-5"}, "datos.xls", opt)

	if strings.Contains(out, "Alarma") {
		t.Error("no debería aparecer la columna Alarma sin límites de alarma")
	}
	if strings.Contains(out, "Fuera de rango") {
		t.Error("no debería informarse de excursiones sin límites de alarma")
	}
	if strings.Contains(out, "Límites de alarma") {
		t.Error("no debería aparecer la línea de límites cuando no se conocen")
	}
}

func TestRenderEmptyBlock(t *testing.T) {
	july := stats.Month{Year: 2025, Month: time.July}
	blocks, err := stats.Split(nil, july, july, 15)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	out := Render(blocks[0], elitech.Device{}, "datos.xls", Options{Decimals: 2, From: july, To: july})
	if !strings.Contains(out, "sin lecturas en este intervalo") {
		t.Errorf("un bloque vacío debería decirlo explícitamente:\n%s", out)
	}
}

func TestFilename(t *testing.T) {
	block, dev, _ := sample(t)

	if got, want := Filename(block, dev), "informe_001_2025-06-01_2025-06-15.txt"; got != want {
		t.Errorf("Filename() = %q, se esperaba %q", got, want)
	}
	if got, want := Filename(block, elitech.Device{}), "informe_2025-06-01_2025-06-15.txt"; got != want {
		t.Errorf("Filename() sin S/N = %q, se esperaba %q", got, want)
	}
	// Un S/N con caracteres problemáticos no debe romper la ruta.
	dirty := elitech.Device{Serial: `A/B:C 1`}
	if got, want := Filename(block, dirty), "informe_ABC-1_2025-06-01_2025-06-15.txt"; got != want {
		t.Errorf("Filename() con S/N sucio = %q, se esperaba %q", got, want)
	}
}

func TestNumUsesCommaDecimal(t *testing.T) {
	cases := []struct {
		v        float64
		decimals int
		want     string
	}{
		{5.3643, 2, "5,36"},
		{-2.7, 1, "-2,7"},
		{19.0, 0, "19"},
		{-0.004, 2, "0,00"}, // no debe salir "-0,00"
	}
	for _, c := range cases {
		if got := num(c.v, c.decimals); got != c.want {
			t.Errorf("num(%v, %d) = %q, se esperaba %q", c.v, c.decimals, got, c.want)
		}
	}
}

func TestWriteFileEncodings(t *testing.T) {
	dir := t.TempDir()
	content := "Máximo 19,2 °C\nsegunda línea\n"

	cases := []struct {
		name   string
		enc    Encoding
		prefix []byte
	}{
		{"utf8bom.txt", UTF8BOM, []byte{0xEF, 0xBB, 0xBF}},
		{"utf8.txt", UTF8, []byte("M")},
		{"cp1252.txt", CP1252, []byte("M")},
	}

	for _, c := range cases {
		if err := WriteFile(dir, c.name, content, c.enc); err != nil {
			t.Fatalf("WriteFile(%s): %v", c.name, err)
		}
		data, err := os.ReadFile(filepath.Join(dir, c.name))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasPrefix(string(data), string(c.prefix)) {
			t.Errorf("%s: no empieza por %q", c.name, c.prefix)
		}
		// Windows espera CRLF, y no deben quedar saltos sueltos.
		if !strings.Contains(string(data), "\r\n") {
			t.Errorf("%s: los saltos de línea deberían ser CRLF", c.name)
		}
		if strings.Contains(strings.ReplaceAll(string(data), "\r\n", ""), "\n") {
			t.Errorf("%s: quedan saltos de línea sin \\r", c.name)
		}
	}

	// En CP1252 los acentos y el grado ocupan un solo byte.
	data, err := os.ReadFile(filepath.Join(dir, "cp1252.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// Se compara sobre los bytes: en CP1252 el ° es 0xB0, que como texto UTF-8
	// no sería una runa válida.
	if !bytes.Contains(data, []byte{0xB0}) {
		t.Error("cp1252: falta el símbolo de grado codificado en un byte")
	}
	if !bytes.Contains(data, []byte{0xE1}) { // á
		t.Error("cp1252: falta la vocal acentuada codificada en un byte")
	}
	if strings.Contains(string(data), "?") {
		t.Error("cp1252: algún carácter no se pudo representar")
	}
}

func TestParseEncoding(t *testing.T) {
	cases := map[string]Encoding{
		"":        UTF8BOM,
		"utf8bom": UTF8BOM,
		"UTF-8":   UTF8,
		"ansi":    CP1252,
		"cp1252":  CP1252,
	}
	for in, want := range cases {
		got, err := ParseEncoding(in)
		if err != nil {
			t.Errorf("ParseEncoding(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseEncoding(%q) = %v, se esperaba %v", in, got, want)
		}
	}
	if _, err := ParseEncoding("ebcdic"); err == nil {
		t.Error("una codificación desconocida debería fallar")
	}
}
