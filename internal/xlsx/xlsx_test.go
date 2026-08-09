package xlsx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dierodfer/temperatureElitechLog/internal/testutil"
	"github.com/dierodfer/temperatureElitechLog/internal/xlsx"
)

func TestOpenInlineStrings(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{
		Name: "Lista",
		Rows: [][]string{
			{"No.", "Tiempo", "Temperatura°C"},
			{"1", "06-01-2021 12:39:33", "19,0"},
			{"2", "06-01-2021 18:39:33", "3,0"},
		},
	})

	book, err := xlsx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	sheet := book.Sheet("Lista")
	if sheet == nil {
		t.Fatal("no se encontró la hoja Lista")
	}
	if got, want := len(sheet.Rows), 3; got != want {
		t.Fatalf("filas = %d, se esperaban %d", got, want)
	}
	if got, want := sheet.Cell(1, 1), "06-01-2021 12:39:33"; got != want {
		t.Errorf("celda B2 = %q, se esperaba %q", got, want)
	}
	if got, want := sheet.Cell(2, 2), "3,0"; got != want {
		t.Errorf("celda C3 = %q, se esperaba %q", got, want)
	}
}

func TestOpenSharedStrings(t *testing.T) {
	path := testutil.WriteXLSXShared(t, testutil.SheetData{
		Name: "Lista",
		Rows: [][]string{
			{"Tiempo", "Temperatura"},
			{"06-01-2021 12:39:33", "19,0"},
			{"07-01-2021 12:39:33", "19,0"}, // repetido: comparte entrada
		},
	})

	book, err := xlsx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sheet := book.Sheet("Lista")
	if got, want := sheet.Cell(1, 1), "19,0"; got != want {
		t.Errorf("celda B2 = %q, se esperaba %q", got, want)
	}
	if got, want := sheet.Cell(2, 0), "07-01-2021 12:39:33"; got != want {
		t.Errorf("celda A3 = %q, se esperaba %q", got, want)
	}
}

// Una celda vacía no debe desplazar las siguientes: la posición la fija la
// referencia de la celda, no el orden de aparición.
func TestGapsKeepColumnPositions(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{
		Name: "Resumen",
		Rows: [][]string{
			{"Modelo", "RC-5", "", "Tipo de sensor", "Temperatura"},
		},
	})

	book, err := xlsx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sheet := book.Sheet("Resumen")
	if got, want := sheet.Cell(0, 3), "Tipo de sensor"; got != want {
		t.Errorf("celda D1 = %q, se esperaba %q", got, want)
	}
	if got := sheet.Cell(0, 2); got != "" {
		t.Errorf("celda C1 = %q, se esperaba vacía", got)
	}
}

func TestSheetLookupIsCaseInsensitive(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{Name: "Lista", Rows: [][]string{{"a"}}})

	book, err := xlsx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if book.Sheet("LISTA") == nil {
		t.Error("la búsqueda de hoja debería ignorar mayúsculas")
	}
	if book.Sheet("Grafico") != nil {
		t.Error("no debería encontrarse una hoja inexistente")
	}
}

func TestCellOutOfRange(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{Name: "Lista", Rows: [][]string{{"a"}}})
	book, err := xlsx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sheet := book.Sheet("Lista")
	for _, c := range []struct{ row, col int }{{-1, 0}, {0, -1}, {9, 9}} {
		if got := sheet.Cell(c.row, c.col); got != "" {
			t.Errorf("Cell(%d,%d) = %q, se esperaba vacía", c.row, c.col, got)
		}
	}
}

// El .xls binario clásico y los HTML renombrados deben dar un error que diga
// qué hacer, no un fallo genérico de ZIP.
func TestOpenRejectsNonOOXML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "antiguo.xls")
	if err := os.WriteFile(path, []byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1datos"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := xlsx.Open(path)
	if err == nil {
		t.Fatal("se esperaba un error al abrir un fichero que no es OOXML")
	}
	if !strings.Contains(err.Error(), "ElitechLog") {
		t.Errorf("el error debería orientar al usuario, fue: %v", err)
	}
}

func TestOpenMissingFile(t *testing.T) {
	if _, err := xlsx.Open(filepath.Join(t.TempDir(), "no-existe.xls")); err == nil {
		t.Fatal("se esperaba un error al abrir un fichero inexistente")
	}
}
