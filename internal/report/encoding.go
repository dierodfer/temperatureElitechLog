package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Encoding es la codificación con la que se vuelca el TXT.
type Encoding int

const (
	// UTF8BOM añade la marca de orden de bytes. Es el valor por defecto porque
	// es lo que hace que el Bloc de notas y Excel muestren bien los acentos y
	// el símbolo de grado en Windows.
	UTF8BOM Encoding = iota
	UTF8
	CP1252
)

// ParseEncoding traduce el valor del flag --codificacion.
func ParseEncoding(s string) (Encoding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "utf8bom", "utf-8-bom", "bom", "":
		return UTF8BOM, nil
	case "utf8", "utf-8":
		return UTF8, nil
	case "cp1252", "ansi", "windows-1252", "latin1":
		return CP1252, nil
	}
	return 0, fmt.Errorf("codificación %q no reconocida: usa utf8bom, utf8 o cp1252", s)
}

// WriteFile vuelca el informe con saltos de línea CRLF, que es lo que espera
// cualquier editor de Windows, y en la codificación indicada.
func WriteFile(dir, name, content string, enc Encoding) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data := encode(toCRLF(content), enc)
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

func toCRLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}

func encode(s string, enc Encoding) []byte {
	switch enc {
	case UTF8:
		return []byte(s)
	case CP1252:
		return toCP1252(s)
	default:
		return append([]byte{0xEF, 0xBB, 0xBF}, s...)
	}
}

// cp1252Extra son los caracteres de Windows-1252 en el rango 0x80-0x9F, que es
// donde esta codificación se aparta de Latin-1. El resto de puntos de código
// por debajo de 0x100 coinciden con su valor en Latin-1.
var cp1252Extra = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87,
	'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E, '‘': 0x91,
	'’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97, '˜': 0x98,
	'™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
}

// toCP1252 convierte a Windows-1252 sustituyendo por '?' lo que no sea
// representable, que con el vocabulario de estos informes no llega a ocurrir.
func toCP1252(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case r >= 0xA0 && r <= 0xFF:
			out = append(out, byte(r))
		default:
			if b, ok := cp1252Extra[r]; ok {
				out = append(out, b)
				continue
			}
			out = append(out, '?')
		}
	}
	return out
}
