package elitech_test

import (
	"testing"
	"time"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
	"github.com/dierodfer/temperatureElitechLog/internal/testutil"
)

// lista devuelve una hoja de datos con el formato exacto que exporta ElitechLog.
func lista(rows ...[]string) testutil.SheetData {
	all := [][]string{{"No.", "Tiempo", "Temperatura°C"}}
	return testutil.SheetData{Name: "Lista", Rows: append(all, rows...)}
}

func TestOpenReadsReadings(t *testing.T) {
	path := testutil.WriteXLSX(t, lista(
		[]string{"1", "06-01-2021 12:39:33", "19,0"},
		[]string{"2", "06-01-2021 18:39:33", "3,0"},
		[]string{"3", "07-01-2021 00:39:33", "-2,7"},
	))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got, want := len(log.Readings), 3; got != want {
		t.Fatalf("lecturas = %d, se esperaban %d", got, want)
	}
	if log.Skipped != 0 {
		t.Errorf("descartadas = %d, se esperaban 0", log.Skipped)
	}

	want := time.Date(2021, 1, 6, 12, 39, 33, 0, time.UTC)
	if !log.Readings[0].Time.Equal(want) {
		t.Errorf("primera fecha = %v, se esperaba %v", log.Readings[0].Time, want)
	}
	if got := log.Readings[2].Temp; got != -2.7 {
		t.Errorf("temperatura negativa = %v, se esperaba -2.7", got)
	}
}

// La fecha 28-06 solo es válida como día-mes, así que el detector debe
// descartar el layout mes-día aunque las primeras filas sean ambiguas.
func TestDetectsDayFirstLayout(t *testing.T) {
	path := testutil.WriteXLSX(t, lista(
		[]string{"1", "06-01-2021 12:39:33", "5,0"},
		[]string{"2", "07-01-2021 12:39:33", "5,0"},
		[]string{"3", "28-06-2026 12:39:33", "4,1"},
	))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got, want := log.TimeLayout, "02-01-2006 15:04:05"; got != want {
		t.Errorf("layout = %q, se esperaba %q", got, want)
	}
	if got, want := log.Readings[0].Time.Month(), time.January; got != want {
		t.Errorf("mes de la primera lectura = %v, se esperaba %v", got, want)
	}
}

func TestParsesISOLayout(t *testing.T) {
	path := testutil.WriteXLSX(t, lista(
		[]string{"1", "2021-01-06 12:39:33", "5,0"},
		[]string{"2", "2021-01-07 12:39:33", "6,0"},
	))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got, want := log.Readings[0].Time.Day(), 6; got != want {
		t.Errorf("día = %d, se esperaba %d", got, want)
	}
}

func TestReadingsAreSorted(t *testing.T) {
	path := testutil.WriteXLSX(t, lista(
		[]string{"1", "09-01-2021 12:39:33", "5,0"},
		[]string{"2", "06-01-2021 12:39:33", "6,0"},
		[]string{"3", "07-01-2021 12:39:33", "7,0"},
	))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for i := 1; i < len(log.Readings); i++ {
		if log.Readings[i].Time.Before(log.Readings[i-1].Time) {
			t.Fatalf("las lecturas no quedaron ordenadas: %v antes de %v",
				log.Readings[i-1].Time, log.Readings[i].Time)
		}
	}
}

func TestSkipsUnreadableRows(t *testing.T) {
	path := testutil.WriteXLSX(t, lista(
		[]string{"1", "06-01-2021 12:39:33", "5,0"},
		[]string{"2", "no es una fecha", "6,0"},
		[]string{"3", "07-01-2021 12:39:33", "N/A"},
		[]string{"4", "08-01-2021 12:39:33", "7,5"},
	))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got, want := len(log.Readings), 2; got != want {
		t.Errorf("lecturas = %d, se esperaban %d", got, want)
	}
	if got, want := log.Skipped, 2; got != want {
		t.Errorf("descartadas = %d, se esperaban %d", got, want)
	}
}

func TestAcceptsDotDecimalAndEnglishHeaders(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{
		Name: "Data",
		Rows: [][]string{
			{"No.", "Date/Time", "Temperature(°C)"},
			{"1", "06-01-2021 12:39:33", "19.5"},
		},
	})

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := log.Readings[0].Temp; got != 19.5 {
		t.Errorf("temperatura = %v, se esperaba 19.5", got)
	}
}

func TestReadsDeviceMetadata(t *testing.T) {
	path := testutil.WriteXLSX(t,
		testutil.SheetData{
			Name: "Resumen",
			Rows: [][]string{
				{"Informe de datos"},
				{"Modelo", "RC-5", "", "Tipo de sensor", "Temperatura"},
				{"S/N", "001", "", "Versión de firmware", "V1,1"},
				{"Presiona el botón", "Inhabilitar", "", "Intervalo", "6h"},
				{"H1: Encima:    8,0°C", "Alarm"},
				{"L1: Abajo:    2,0°C", "Alarm"},
			},
		},
		lista([]string{"1", "06-01-2021 12:39:33", "5,0"}),
	)

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	dev := log.Device
	if dev.Model != "RC-5" {
		t.Errorf("modelo = %q, se esperaba RC-5", dev.Model)
	}
	if dev.Serial != "001" {
		t.Errorf("S/N = %q, se esperaba 001", dev.Serial)
	}
	if dev.Sensor != "Temperatura" {
		t.Errorf("sensor = %q, se esperaba Temperatura", dev.Sensor)
	}
	if dev.Interval != "6h" {
		t.Errorf("intervalo = %q, se esperaba 6h", dev.Interval)
	}
	if dev.HighLimit == nil || *dev.HighLimit != 8 {
		t.Errorf("H1 = %v, se esperaba 8", dev.HighLimit)
	}
	if dev.LowLimit == nil || *dev.LowLimit != 2 {
		t.Errorf("L1 = %v, se esperaba 2", dev.LowLimit)
	}
}

// Sin hoja Resumen el informe debe seguir generándose, solo que sin metadatos.
func TestMissingMetadataIsNotFatal(t *testing.T) {
	path := testutil.WriteXLSX(t, lista([]string{"1", "06-01-2021 12:39:33", "5,0"}))

	log, err := elitech.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if log.Device.HighLimit != nil || log.Device.LowLimit != nil {
		t.Error("no debería haber límites de alarma sin hoja Resumen")
	}
}

func TestErrorsWithoutDataSheet(t *testing.T) {
	path := testutil.WriteXLSX(t, testutil.SheetData{
		Name: "Grafico",
		Rows: [][]string{{"solo", "una", "imagen"}},
	})

	if _, err := elitech.Open(path); err == nil {
		t.Fatal("se esperaba un error al no haber columnas de tiempo y temperatura")
	}
}

func TestErrorsWithoutRows(t *testing.T) {
	path := testutil.WriteXLSX(t, lista())
	if _, err := elitech.Open(path); err == nil {
		t.Fatal("se esperaba un error al no haber ninguna lectura")
	}
}
