package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func touch(t *testing.T, path string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestFindRecordsNewestFirst(t *testing.T) {
	base := t.TempDir()
	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.Local)
	touch(t, filepath.Join(base, "registros", "viejo.xls"), now.Add(-48*time.Hour))
	touch(t, filepath.Join(base, "registros", "nuevo.XLSX"), now)
	touch(t, filepath.Join(base, "registros", "medio.xls"), now.Add(-24*time.Hour))
	touch(t, filepath.Join(base, "registros", "notas.txt"), now.Add(time.Hour))
	touch(t, filepath.Join(base, "registros", "~$nuevo.xlsx"), now.Add(time.Hour))
	touch(t, filepath.Join(base, "suelto.xls"), now.Add(time.Hour)) // fuera de registros

	got := findRecords(base)

	want := []string{"nuevo.XLSX", "medio.xls", "viejo.xls"}
	if len(got) != len(want) {
		t.Fatalf("ficheros = %d, se esperaban %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Name != w {
			t.Errorf("posición %d = %s, se esperaba %s", i, got[i].Name, w)
		}
	}
	if !got[0].ModTime.Equal(now) {
		t.Errorf("fecha = %v, se esperaba %v", got[0].ModTime, now)
	}
}

func TestFindRecordsMergesBasesWithoutDuplicates(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	now := time.Now()
	touch(t, filepath.Join(a, "registros", "a.xls"), now.Add(-time.Hour))
	touch(t, filepath.Join(b, "registros", "b.xls"), now)

	got := findRecords(a, a, "", b)
	if len(got) != 2 || got[0].Name != "b.xls" {
		t.Errorf("resultado inesperado: %v", got)
	}
}

func TestFindRecordsMissingFolder(t *testing.T) {
	if got := findRecords(t.TempDir()); len(got) != 0 {
		t.Errorf("sin carpeta registros no debería haber ficheros: %v", got)
	}
}

func TestPickRecord(t *testing.T) {
	files := []recordFile{{Path: "registros/a.xls"}, {Path: "registros/b.xls"}}
	cases := map[string]string{
		"2":               "registros/b.xls",
		" 1 ":             "registros/a.xls",
		"3":               "3", // fuera de la lista: se trata como ruta
		"0":               "0",
		"otro/c.xls":      "otro/c.xls",
		"registros/a.xls": "registros/a.xls",
	}
	for in, want := range cases {
		if got := pickRecord(in, files); got != want {
			t.Errorf("pickRecord(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}
