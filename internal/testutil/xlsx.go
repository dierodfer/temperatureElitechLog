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
	shared := &sharedStrings{index: map[string]int{}}

	var wbSheets, rels strings.Builder
	for i, sh := range sheets {
		rid := fmt.Sprintf("rId%d", i+1)
		fmt.Fprintf(&wbSheets, `<sheet name=%s sheetId="%d" r:id="%s"/>`, attr(sh.Name), i+1, rid)
		fmt.Fprintf(&rels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, rid, i+1)
		add(t, zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), sheetXML(sh, useShared, shared))
	}

	add(t, zw, "[Content_Types].xml", contentTypesXML)
	add(t, zw, "xl/workbook.xml", workbookXML(wbSheets.String()))
	add(t, zw, "xl/_rels/workbook.xml.rels", workbookRelsXML(rels.String()))
	if useShared {
		add(t, zw, "xl/sharedStrings.xml", sharedStringsXML(shared.values))
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("cerrando el zip de prueba: %v", err)
	}
	return path
}

// sharedStrings interna las cadenas usadas en modo sharedStrings, asignando a
// cada una un índice estable la primera vez que aparece.
type sharedStrings struct {
	values []string
	index  map[string]int
}

func (s *sharedStrings) intern(v string) int {
	if i, ok := s.index[v]; ok {
		return i
	}
	i := len(s.values)
	s.values = append(s.values, v)
	s.index[v] = i
	return i
}

// sheetXML genera el XML de una hoja completa.
func sheetXML(sh SheetData, useShared bool, shared *sharedStrings) string {
	var body strings.Builder
	for r, row := range sh.Rows {
		fmt.Fprintf(&body, `<row r="%d">`, r+1)
		for c, cell := range row {
			if cell != "" { // hueco: el lector debe respetar la posición
				body.WriteString(cellXML(c, r, cell, useShared, shared))
			}
		}
		body.WriteString(`</row>`)
	}
	return `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
		body.String() + `</sheetData></worksheet>`
}

// cellXML genera el XML de una única celda no vacía, como inline string o
// como referencia a sharedStrings según useShared.
func cellXML(col, row int, value string, useShared bool, shared *sharedStrings) string {
	ref := fmt.Sprintf("%s%d", colName(col), row+1)
	if useShared {
		return fmt.Sprintf(`<c r="%s" t="s"><v>%d</v></c>`, ref, shared.intern(value))
	}
	return fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, escape(value))
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`

func workbookXML(sheetsXML string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` +
		sheetsXML + `</sheets></workbook>`
}

func workbookRelsXML(relsXML string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		relsXML + `</Relationships>`
}

func sharedStringsXML(values []string) string {
	var si strings.Builder
	for _, s := range values {
		fmt.Fprintf(&si, `<si><t>%s</t></si>`, escape(s))
	}
	return `<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		si.String() + `</sst>`
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
