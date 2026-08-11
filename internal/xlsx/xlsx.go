// Package xlsx implementa un lector mínimo de ficheros OOXML (.xlsx) usando
// únicamente la librería estándar. Solo cubre lo necesario para leer las hojas
// que exporta ElitechLog: celdas de texto (inline o compartidas) y numéricas.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

// magic son los primeros bytes de cualquier fichero ZIP, y por tanto de cualquier XLSX.
var magic = []byte{'P', 'K', 0x03, 0x04}

// Sheet es una hoja de cálculo ya materializada: una fila por elemento, y en
// cada fila una celda por columna. Las celdas ausentes quedan como cadena vacía.
type Sheet struct {
	Name string
	Rows [][]string
}

// Cell devuelve el contenido de una celda tolerando índices fuera de rango, que
// en una hoja irregular son la norma y no un error.
func (s *Sheet) Cell(row, col int) string {
	if row < 0 || row >= len(s.Rows) {
		return ""
	}
	if col < 0 || col >= len(s.Rows[row]) {
		return ""
	}
	return s.Rows[row][col]
}

// File es un libro de trabajo abierto en memoria.
type File struct {
	Sheets []*Sheet
}

// Sheet busca una hoja por nombre, ignorando mayúsculas y espacios sobrantes.
func (f *File) Sheet(name string) *Sheet {
	for _, s := range f.Sheets {
		if strings.EqualFold(strings.TrimSpace(s.Name), strings.TrimSpace(name)) {
			return s
		}
	}
	return nil
}

// Open lee un libro completo desde disco.
//
// ElitechLog 8.0 exporta ficheros OOXML con extensión .xls, así que el formato
// se decide por los bytes iniciales y nunca por la extensión.
func Open(filename string) (*File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	head := make([]byte, len(magic))
	if _, err := io.ReadFull(f, head); err != nil || string(head) != string(magic) {
		return nil, fmt.Errorf("%s no es un fichero XLSX (OOXML): parece un .xls binario "+
			"antiguo o un HTML renombrado; vuelve a exportarlo desde ElitechLog en formato Excel",
			path.Base(filename))
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %s como XLSX: %w", path.Base(filename), err)
	}
	return read(zr)
}

func read(zr *zip.Reader) (*File, error) {
	files := make(map[string]*zip.File, len(zr.File))
	for _, zf := range zr.File {
		files[strings.TrimPrefix(zf.Name, "/")] = zf
	}

	shared, err := readSharedStrings(files)
	if err != nil {
		return nil, err
	}
	rels, err := readRels(files)
	if err != nil {
		return nil, err
	}
	entries, err := readWorkbook(files)
	if err != nil {
		return nil, err
	}

	out := &File{}
	for i, e := range entries {
		target := rels[e.RID]
		if target == "" {
			// Sin relación explícita, XLSX usa el orden de declaración.
			target = fmt.Sprintf("worksheets/sheet%d.xml", i+1)
		}
		zf := files[path.Join("xl", target)]
		if zf == nil {
			if zf = files[target]; zf == nil {
				continue
			}
		}
		rows, err := readSheet(zf, shared)
		if err != nil {
			return nil, fmt.Errorf("hoja %q: %w", e.Name, err)
		}
		out.Sheets = append(out.Sheets, &Sheet{Name: e.Name, Rows: rows})
	}
	if len(out.Sheets) == 0 {
		return nil, fmt.Errorf("el fichero no contiene ninguna hoja legible")
	}
	return out, nil
}

type sheetEntry struct {
	Name string
	RID  string
}

func readWorkbook(files map[string]*zip.File) ([]sheetEntry, error) {
	zf := files["xl/workbook.xml"]
	if zf == nil {
		return nil, fmt.Errorf("falta xl/workbook.xml: el fichero no es un libro de Excel válido")
	}
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var doc struct {
		Sheets struct {
			Sheet []struct {
				Name string `xml:"name,attr"`
				RID  string `xml:"id,attr"`
			} `xml:"sheet"`
		} `xml:"sheets"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, fmt.Errorf("xl/workbook.xml ilegible: %w", err)
	}

	entries := make([]sheetEntry, 0, len(doc.Sheets.Sheet))
	for _, s := range doc.Sheets.Sheet {
		entries = append(entries, sheetEntry{Name: s.Name, RID: s.RID})
	}
	return entries, nil
}

func readRels(files map[string]*zip.File) (map[string]string, error) {
	rels := map[string]string{}
	zf := files["xl/_rels/workbook.xml.rels"]
	if zf == nil {
		return rels, nil
	}
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var doc struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, fmt.Errorf("xl/_rels/workbook.xml.rels ilegible: %w", err)
	}
	for _, r := range doc.Rel {
		rels[r.ID] = strings.TrimPrefix(r.Target, "/xl/")
	}
	return rels, nil
}

func readSharedStrings(files map[string]*zip.File) ([]string, error) {
	zf := files["xl/sharedStrings.xml"]
	if zf == nil {
		return nil, nil
	}
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var doc struct {
		SI []struct {
			T string   `xml:"t"`
			R []string `xml:"r>t"`
		} `xml:"si"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, fmt.Errorf("xl/sharedStrings.xml ilegible: %w", err)
	}

	out := make([]string, len(doc.SI))
	for i, si := range doc.SI {
		if len(si.R) > 0 {
			// Texto enriquecido: la cadena se reparte en varios fragmentos.
			out[i] = strings.Join(si.R, "")
			continue
		}
		out[i] = si.T
	}
	return out, nil
}

// sheetState es el estado acumulado al recorrer una hoja en modo streaming.
// Separarlo en un tipo con métodos —uno por tipo de token XML— reparte la
// lógica que antes vivía en un único bucle con switches anidados.
type sheetState struct {
	shared  []string
	rows    [][]string
	row     []string
	col     int
	inRow   bool
	cellTyp string
	cellRef string
	value   strings.Builder
	capture bool
}

// readSheet recorre la hoja en modo streaming: sheet2.xml del fichero de
// ejemplo ocupa casi 2 MB, y decodificarlo entero de golpe no aporta nada.
func readSheet(zf *zip.File, shared []string) ([][]string, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	st := &sheetState{shared: shared}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			st.start(t)
		case xml.CharData:
			st.char(t)
		case xml.EndElement:
			st.end(t)
		}
	}
	return st.rows, nil
}

func (st *sheetState) start(t xml.StartElement) {
	switch t.Name.Local {
	case "row":
		st.inRow, st.row, st.col = true, nil, 0
	case "c":
		if st.inRow {
			st.startCell(t)
		}
	case "v", "t":
		st.capture = true
	}
}

// startCell procesa la apertura de una celda: fija su tipo, su columna real
// (a partir de la referencia, para no desplazarse con huecos) y prepara el
// acumulador de su valor.
func (st *sheetState) startCell(t xml.StartElement) {
	st.cellTyp, st.cellRef = "", ""
	for _, a := range t.Attr {
		switch a.Name.Local {
		case "t":
			st.cellTyp = a.Value
		case "r":
			st.cellRef = a.Value
		}
	}
	// La referencia manda: así una celda vacía no desplaza el resto.
	if idx, ok := columnIndex(st.cellRef); ok {
		st.col = idx
	}
	st.value.Reset()
}

func (st *sheetState) char(t xml.CharData) {
	if st.capture {
		st.value.Write(t)
	}
}

func (st *sheetState) end(t xml.EndElement) {
	switch t.Name.Local {
	case "v", "t":
		st.capture = false
	case "c":
		if st.inRow {
			st.row = setAt(st.row, st.col, decodeValue(st.value.String(), st.cellTyp, st.shared))
			st.col++
		}
	case "row":
		st.rows = append(st.rows, st.row)
		st.inRow = false
	}
}

// decodeValue traduce el contenido bruto de una celda según su tipo declarado.
// Los valores numéricos y las fechas se dejan tal cual: quien interpreta el
// dominio es internal/elitech, no el lector.
func decodeValue(raw, typ string, shared []string) string {
	switch typ {
	case "s":
		i, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || i < 0 || i >= len(shared) {
			return ""
		}
		return shared[i]
	default:
		return raw
	}
}

// setAt coloca un valor en una posición concreta, ampliando la fila con celdas
// vacías si hace falta.
func setAt(row []string, idx int, val string) []string {
	for len(row) <= idx {
		row = append(row, "")
	}
	row[idx] = val
	return row
}

// columnIndex convierte la parte alfabética de una referencia ("C7998") en un
// índice de columna base 0.
func columnIndex(ref string) (int, bool) {
	n := 0
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		switch {
		case c >= 'A' && c <= 'Z':
			n = n*26 + int(c-'A') + 1
		case c >= 'a' && c <= 'z':
			n = n*26 + int(c-'a') + 1
		case c >= '0' && c <= '9':
			if n == 0 {
				return 0, false
			}
			return n - 1, true
		default:
			return 0, false
		}
	}
	if n == 0 {
		return 0, false
	}
	return n - 1, true
}
