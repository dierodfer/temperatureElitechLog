package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/dierodfer/temperatureElitechLog/internal/app"
	"github.com/dierodfer/temperatureElitechLog/internal/testutil"
)

// fixture escribe un libro con lecturas cada 6 h entre dos días de junio 2025.
func fixture(t *testing.T, rows ...[]string) string {
	t.Helper()
	data := [][]string{{"No.", "Tiempo", "Temperatura°C"}}
	return testutil.WriteXLSX(t,
		testutil.SheetData{Name: "Resumen", Rows: [][]string{
			{"Modelo", "RC-5", "", "Tipo de sensor", "Temperatura"},
			{"S/N", "001"},
			{"H1: Encima:    8,0°C", "Alarm"},
			{"L1: Abajo:    2,0°C", "Alarm"},
		}},
		testutil.SheetData{Name: "Lista", Rows: append(data, rows...)},
	)
}

func juneRows() [][]string {
	var rows [][]string
	n := 0
	for day := 1; day <= 30; day++ {
		for _, hour := range []string{"00", "06", "12", "18"} {
			n++
			rows = append(rows, []string{
				string(rune('0' + n%10)),
				pad2(day) + "-06-2025 " + hour + ":39:33",
				"5,0",
			})
		}
	}
	return rows
}

func pad2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestGenerateWritesOneFilePerBlock(t *testing.T) {
	in := fixture(t, juneRows()...)
	out := t.TempDir()

	res, err := app.Generate(app.Request{
		Input: in, From: "2025-06", To: "2025-06",
		Interval: 15, OutputDir: out, Decimals: 2, Encoding: "utf8bom",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if got, want := res.Blocks, 2; got != want {
		t.Errorf("bloques = %d, se esperaban %d", got, want)
	}
	if got, want := res.Readings, 120; got != want {
		t.Errorf("lecturas = %d, se esperaban %d", got, want)
	}

	want := []string{
		"informe_001_2025-06-01_2025-06-15.txt",
		"informe_001_2025-06-16_2025-06-30.txt",
	}
	got := listDir(t, out)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ficheros = %v, se esperaban %v", got, want)
	}

	data, err := os.ReadFile(filepath.Join(out, want[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("el informe debería llevar BOM UTF-8 por defecto")
	}
	if !strings.Contains(string(data), "RC-5") {
		t.Error("el informe debería incluir el modelo del dispositivo")
	}
}

// Los meses sin lecturas no generan fichero salvo que se pidan expresamente.
func TestGenerateSkipsEmptyBlocks(t *testing.T) {
	in := fixture(t, juneRows()...)

	out := t.TempDir()
	res, err := app.Generate(app.Request{
		Input: in, From: "2025-06", To: "2025-07",
		Interval: 15, OutputDir: out, Decimals: 2, Encoding: "utf8bom",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, want := res.Blocks, 2; got != want {
		t.Errorf("bloques generados = %d, se esperaban %d", got, want)
	}
	if got, want := res.Skipped, 2; got != want {
		t.Errorf("bloques omitidos = %d, se esperaban %d", got, want)
	}
	if len(res.Warnings) == 0 {
		t.Error("se esperaba un aviso por pedir meses fuera de los datos")
	}

	out = t.TempDir()
	res, err = app.Generate(app.Request{
		Input: in, From: "2025-06", To: "2025-07", IncludeEmpty: true,
		Interval: 15, OutputDir: out, Decimals: 2, Encoding: "utf8bom",
	})
	if err != nil {
		t.Fatalf("Generate con --incluir-vacios: %v", err)
	}
	if got, want := res.Blocks, 4; got != want {
		t.Errorf("bloques con --incluir-vacios = %d, se esperaban %d", got, want)
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	in := fixture(t, juneRows()...)
	base := app.Request{
		Input: in, From: "2025-06", To: "2025-06",
		Interval: 15, OutputDir: t.TempDir(), Decimals: 2, Encoding: "utf8bom",
	}

	cases := map[string]func(r *app.Request){
		"mes inicial inválido":  func(r *app.Request) { r.From = "junio" },
		"mes final inválido":    func(r *app.Request) { r.To = "" },
		"rango invertido":       func(r *app.Request) { r.From, r.To = "2025-07", "2025-06" },
		"intervalo cero":        func(r *app.Request) { r.Interval = 0 },
		"decimales negativos":   func(r *app.Request) { r.Decimals = -1 },
		"codificación inválida": func(r *app.Request) { r.Encoding = "ebcdic" },
		"fichero inexistente":   func(r *app.Request) { r.Input = "no-existe.xls" },
		"rango sin lecturas":    func(r *app.Request) { r.From, r.To = "2030-01", "2030-02" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := base
			mutate(&req)
			if _, err := app.Generate(req); err == nil {
				t.Error("se esperaba un error")
			}
		})
	}
}

func TestResultPrint(t *testing.T) {
	res := &app.Result{
		Files:     []string{"informe_001_2025-06-01_2025-06-15.txt"},
		OutputDir: "informes", Blocks: 1, Skipped: 2, Readings: 120,
		Warnings: []string{"aviso de prueba"},
	}

	var buf bytes.Buffer
	res.Print(&buf)
	out := buf.String()

	for _, want := range []string{"120", "informe_001_2025-06-01_2025-06-15.txt", "aviso de prueba", "--incluir-vacios"} {
		if !strings.Contains(out, want) {
			t.Errorf("el resumen debería mencionar %q:\n%s", want, out)
		}
	}
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}
