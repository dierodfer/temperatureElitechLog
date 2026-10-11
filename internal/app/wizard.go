package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
	"github.com/dierodfer/temperatureElitechLog/internal/stats"
)

// Wizard guía al usuario paso a paso. Está pensado para cuando el ejecutable
// se abre con doble clic en Windows, donde no hay forma de pasar flags.
func Wizard(version string) error {
	in := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("===============================================")
	fmt.Println("  Informes de temperatura - ElitechLog")
	fmt.Println("  version", version)
	fmt.Println("===============================================")
	fmt.Println()
	fmt.Println("Pulsa Intro para aceptar el valor entre corchetes.")
	fmt.Println()

	// Al terminar hay que dejar la ventana abierta: si el proceso muere, el
	// Explorador la cierra al instante y no da tiempo a leer nada.
	defer waitForEnter(in)

	input, log, err := askFile(in)
	if err != nil {
		return err
	}

	first := stats.MonthOf(log.Readings[0].Time)
	last := stats.MonthOf(log.Readings[len(log.Readings)-1].Time)

	fmt.Printf("\n  Dispositivo : %s\n", describe(log.Device))
	fmt.Printf("  Lecturas    : %d\n", len(log.Readings))
	fmt.Printf("  Periodo     : %s a %s\n\n", first.Human(), last.Human())

	from := askMonth(in, "Mes inicial (AAAA-MM)", first)
	to := askMonth(in, "Mes final   (AAAA-MM)", last)
	interval := askInt(in, "Días por informe", 15)
	outDir := askText(in, "Carpeta de salida", "informes")

	fmt.Println("\nGenerando informes...")

	res, err := Generate(Request{
		Input:     input,
		From:      from.String(),
		To:        to.String(),
		Interval:  interval,
		OutputDir: outDir,
		Decimals:  2,
		Encoding:  "utf8bom",
	})
	if err != nil {
		return err
	}
	res.Print(os.Stdout)
	return nil
}

// askFile pide el fichero hasta que uno se deja leer, en lugar de abortar al
// primer intento fallido: teclear rutas en Windows es propenso a erratas.
func askFile(in *bufio.Reader) (string, *elitech.Log, error) {
	recent := findRecords(searchBases()...)
	if len(recent) > maxListed {
		recent = recent[:maxListed]
	}
	suggestion := ""
	if len(recent) > 0 {
		suggestion = recent[0].Path
		printRecords(recent)
	} else {
		suggestion = findLocalFile()
	}

	for attempt := 0; ; attempt++ {
		answer := unquote(askText(in, "Fichero exportado por ElitechLog (ruta o número)", suggestion))
		path := pickRecord(answer, recent)
		if path == "" {
			fmt.Println("  Debes indicar un fichero.")
			continue
		}

		log, err := elitech.Open(path)
		if err == nil {
			return path, log, nil
		}

		fmt.Printf("  No se pudo leer: %v\n", err)
		if attempt >= 4 {
			return "", nil, err
		}
		suggestion = ""
		recent = nil
	}
}

// printRecords enseña los ficheros de la carpeta registros con su fecha de
// modificación, para que se vea de un vistazo cuál es el último exportado.
func printRecords(files []recordFile) {
	fmt.Printf("Ficheros en la carpeta %q (el más reciente primero):\n", recordsDir)
	for i, f := range files {
		note := ""
		if i == 0 {
			note = "  <- más reciente"
		}
		fmt.Printf("  %d) %s   modificado %s%s\n", i+1, f.Name, f.ModTime.Format(modFmt), note)
	}
	fmt.Println()
}

// findLocalFile propone el único .xls/.xlsx del directorio actual, que es el
// caso habitual cuando se deja el ejecutable junto al fichero exportado.
func findLocalFile() string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}
	var found string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".xls", ".xlsx":
			if found != "" {
				return "" // hay varios: mejor no elegir por el usuario
			}
			found = e.Name()
		}
	}
	return found
}

func askText(in *bufio.Reader, prompt, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", prompt, def)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func askMonth(in *bufio.Reader, prompt string, def stats.Month) stats.Month {
	for {
		raw := askText(in, prompt, def.String())
		m, err := stats.ParseMonth(raw)
		if err == nil {
			return m
		}
		fmt.Println("  " + err.Error())
	}
}

func askInt(in *bufio.Reader, prompt string, def int) int {
	for {
		raw := askText(in, prompt, strconv.Itoa(def))
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err == nil && n >= 1 {
			return n
		}
		fmt.Println("  Escribe un número entero de días mayor que cero.")
	}
}

// unquote limpia las comillas que añade el Explorador de Windows al arrastrar
// un fichero a la consola.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	return strings.TrimSpace(s)
}

func describe(d elitech.Device) string {
	switch {
	case d.Model != "" && d.Serial != "":
		return d.Model + " (S/N " + d.Serial + ")"
	case d.Model != "":
		return d.Model
	case d.Serial != "":
		return "S/N " + d.Serial
	}
	return "desconocido"
}

func waitForEnter(in *bufio.Reader) {
	fmt.Print("\nPulsa Intro para salir...")
	_, _ = in.ReadString('\n')
	fmt.Println()
}
