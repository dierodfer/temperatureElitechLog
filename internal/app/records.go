package app

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// recordsDir es la carpeta, junto al ejecutable, donde se dejan los ficheros
// exportados por ElitechLog.
const recordsDir = "registros"

// maxListed limita cuántos ficheros recientes se enseñan en el asistente.
const maxListed = 5

const modFmt = "02-01-2006 15:04"

// recordFile es un fichero exportado encontrado en la carpeta de registros.
type recordFile struct {
	Path    string // ruta lista para abrir
	Name    string
	ModTime time.Time
}

// findRecords busca ficheros .xls/.xlsx en la subcarpeta registros de cada base
// y los devuelve del más reciente al más antiguo según su fecha de modificación.
// Una base repetida o una carpeta inexistente no son un error: simplemente no
// aportan nada.
func findRecords(bases ...string) []recordFile {
	var out []recordFile
	seen := map[string]bool{}

	for _, base := range bases {
		if base == "" {
			continue
		}
		dir := filepath.Join(base, recordsDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !isExport(e.Name()) {
				continue
			}
			full := filepath.Join(dir, e.Name())
			abs, err := filepath.Abs(full)
			if err != nil || seen[abs] {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			seen[abs] = true
			out = append(out, recordFile{Path: full, Name: e.Name(), ModTime: info.ModTime()})
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out
}

// isExport descarta lo que no es un Excel y los temporales de bloqueo que crea
// Excel mientras un libro está abierto (~$nombre.xlsx).
func isExport(name string) bool {
	if strings.HasPrefix(name, "~$") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".xls", ".xlsx":
		return true
	}
	return false
}

// searchBases son las carpetas donde puede estar «registros»: la actual, que es
// donde se lanza el ejecutable desde una consola, y la del propio ejecutable,
// que es la que cuenta al abrirlo con doble clic.
func searchBases() []string {
	bases := []string{"."}
	if exe, err := os.Executable(); err == nil {
		bases = append(bases, filepath.Dir(exe))
	}
	return bases
}

// pickRecord interpreta la respuesta del usuario: un número de la lista
// elige ese fichero y cualquier otra cosa se deja tal cual como ruta.
func pickRecord(answer string, files []recordFile) string {
	if n, err := strconv.Atoi(strings.TrimSpace(answer)); err == nil && n >= 1 && n <= len(files) {
		return files[n-1].Path
	}
	return answer
}
