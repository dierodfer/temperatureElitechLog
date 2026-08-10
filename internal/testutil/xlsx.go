// Package testutil construye libros XLSX de prueba.
//
// Los ficheros se generan en tiempo de test en lugar de commitear binarios:
// así el contenido de cada caso se lee en el propio test y no hay datos reales
// del usuario en el repositorio.
package testutil

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SheetData es una hoja de prueba: su nombre y sus filas.
type SheetData struct {
	Name string
	Rows [][]string
}

// WriteXLSX crea un libro XLSX con las hojas indicadas y devuelve su ruta.
// Las celdas se escriben como inline strings, igual que hace ElitechLog.
func WriteXLSX(t *testing.T, sheets ...SheetData) string {
	t.Helper()
	return writeXLSX(t, false, sheets...)
}

// WriteXLSXShared hace lo mismo pero volcando el texto a sharedStrings, que es
// como lo escriben Excel y la mayoría de librerías.
func WriteXLSXShared(t *testing.T, sheets ...SheetData) string {
	t.Helper()
	return writeXLSX(t, true, sheets...)
}

func writeXLSX(t *testing.T, useShared bool, sheets ...SheetData) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "libro.xls")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creando el fichero de prueba: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	var (
		shared    []string
		sharedIdx = map[string]int{}
	)
	intern := func(s string) int {
		if i, ok := sharedIdx[s]; ok {
			return i
		}
		i := len(shared)
		shared = append(shared, s)
		sharedIdx[s] = i
		return i
	}

	var wbSheets, rels strings.Builder
	for i, sh := range sheets {
		rid := fmt.Sprintf("rId%d", i+1)
		fmt.Fprintf(&wbSheets, `<sheet name=%s sheetId="%d" r:id="%s"/>`, attr(sh.Name), i+1, rid)
		fmt.Fprintf(&rels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, rid, i+1)

		var body strings.Builder
		for r, row := range sh.Rows {
			fmt.Fprintf(&body, `<row r="%d">`, r+1)
			for c, cell := range row {
				if cell == "" {
					continue // hueco: el lector debe respetar la posición
				}
				ref := fmt.Sprintf("%s%d", colName(c), r+1)
				if useShared {
					fmt.Fprintf(&body, `<c r="%s" t="s"><v>%d</v></c>`, ref, intern(cell))
					continue
				}
				fmt.Fprintf(&body, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, escape(cell))
			}
			body.WriteString(`</row>`)
		}

		add(t, zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1),
			`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`+
				body.String()+`</sheetData></worksheet>`)
	}

	add(t, zw, "[Content_Types].xml",
		`<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`)
	add(t, zw, "xl/workbook.xml",
		`<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`+
			wbSheets.String()+`</sheets></workbook>`)
	add(t, zw, "xl/_rels/workbook.xml.rels",
		`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
			rels.String()+`</Relationships>`)

	if useShared {
		var si strings.Builder
		for _, s := range shared {
			fmt.Fprintf(&si, `<si><t>%s</t></si>`, escape(s))
		}
		add(t, zw, "xl/sharedStrings.xml",
			`<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`+
				si.String()+`</sst>`)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("cerrando el zip de prueba: %v", err)
	}
	return path
}

func add(t *testing.T, zw *zip.Writer, name, content string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("añadiendo %s: %v", name, err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("escribiendo %s: %v", name, err)
	}
}

func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func attr(s string) string { return `"` + escape(s) + `"` }

func colName(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}
