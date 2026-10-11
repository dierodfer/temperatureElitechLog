package stats

import (
	"math"
	"testing"
	"time"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
)

func at(day int, hour int, temp float64) elitech.Reading {
	return elitech.Reading{
		Time: time.Date(2025, 6, day, hour, 39, 33, 0, time.UTC),
		Temp: temp,
	}
}

func TestByDayAggregates(t *testing.T) {
	days := ByDay([]elitech.Reading{
		at(1, 0, 3.0),
		at(1, 6, 5.0),
		at(1, 12, 10.0),
		at(2, 0, 4.0),
	}, nil, nil)

	if got, want := len(days), 2; got != want {
		t.Fatalf("días = %d, se esperaban %d", got, want)
	}

	d := days[0]
	if d.Count != 3 {
		t.Errorf("lecturas del día 1 = %d, se esperaban 3", d.Count)
	}
	if math.Abs(d.Mean-6.0) > 1e-9 {
		t.Errorf("media = %v, se esperaba 6", d.Mean)
	}
	if d.Min != 3.0 || d.Max != 10.0 {
		t.Errorf("min/max = %v/%v, se esperaban 3/10", d.Min, d.Max)
	}
	if got, want := d.MinAt.Hour(), 0; got != want {
		t.Errorf("hora del mínimo = %d, se esperaba %d", got, want)
	}
	if got, want := d.MaxAt.Hour(), 12; got != want {
		t.Errorf("hora del máximo = %d, se esperaba %d", got, want)
	}
}

func TestByDayKeepsIndividualReadings(t *testing.T) {
	days := ByDay([]elitech.Reading{at(1, 12, 7), at(1, 0, 3), at(2, 6, 4), at(1, 6, 5)}, nil, nil)

	got := days[0].Readings
	if len(got) != 3 {
		t.Fatalf("lecturas del día 1 = %d, se esperaban 3", len(got))
	}
	for i, wantHour := range []int{0, 6, 12} {
		if got[i].Time.Hour() != wantHour {
			t.Errorf("lectura %d a las %d h, se esperaba a las %d h", i, got[i].Time.Hour(), wantHour)
		}
	}
	if len(days[1].Readings) != 1 {
		t.Errorf("lecturas del día 2 = %d, se esperaba 1", len(days[1].Readings))
	}
}

func TestByDaySortsChronologically(t *testing.T) {
	days := ByDay([]elitech.Reading{at(9, 0, 1), at(2, 0, 1), at(5, 0, 1)}, nil, nil)
	for i := 1; i < len(days); i++ {
		if !days[i-1].Date.Before(days[i].Date) {
			t.Fatalf("días desordenados: %v antes de %v", days[i-1].Date, days[i].Date)
		}
	}
}

func TestByDayCountsExcursions(t *testing.T) {
	high, low := 8.0, 2.0
	days := ByDay([]elitech.Reading{
		at(1, 0, 1.5),  // por debajo
		at(1, 6, 5.0),  // dentro
		at(1, 12, 9.0), // por encima
		at(1, 18, 8.0), // justo en el límite: no es excursión
	}, &high, &low)

	if got, want := days[0].Below, 1; got != want {
		t.Errorf("lecturas por debajo = %d, se esperaban %d", got, want)
	}
	if got, want := days[0].Above, 1; got != want {
		t.Errorf("lecturas por encima = %d, se esperaban %d", got, want)
	}
}

func TestByDayWithoutLimitsCountsNothing(t *testing.T) {
	days := ByDay([]elitech.Reading{at(1, 0, 100)}, nil, nil)
	if days[0].Above != 0 || days[0].Below != 0 {
		t.Error("sin límites de alarma no debería contarse ninguna excursión")
	}
}

func TestParseMonth(t *testing.T) {
	cases := []struct {
		in   string
		want Month
	}{
		{"2025-06", Month{2025, time.June}},
		{"2025/06", Month{2025, time.June}},
		{"06-2025", Month{2025, time.June}},
		{" 2026-1 ", Month{2026, time.January}},
	}
	for _, c := range cases {
		got, err := ParseMonth(c.in)
		if err != nil {
			t.Errorf("ParseMonth(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseMonth(%q) = %v, se esperaba %v", c.in, got, c.want)
		}
	}

	for _, bad := range []string{"", "junio", "2025-13", "2025"} {
		if _, err := ParseMonth(bad); err == nil {
			t.Errorf("ParseMonth(%q) debería fallar", bad)
		}
	}
}

func TestMonthHuman(t *testing.T) {
	if got, want := (Month{2025, time.June}).Human(), "junio 2025"; got != want {
		t.Errorf("Human() = %q, se esperaba %q", got, want)
	}
}

// Las quincenas naturales cortan en 1-15 y 16-fin de mes, sea cual sea la
// longitud del mes, y nunca cruzan el cambio de mes.
func TestSplitNaturalFortnights(t *testing.T) {
	cases := []struct {
		name  string
		month Month
		want  [][2]string
	}{
		{"mes de 30 días", Month{2025, time.June}, [][2]string{
			{"2025-06-01", "2025-06-15"}, {"2025-06-16", "2025-06-30"},
		}},
		{"mes de 31 días", Month{2025, time.July}, [][2]string{
			{"2025-07-01", "2025-07-15"}, {"2025-07-16", "2025-07-31"},
		}},
		{"febrero de 28 días", Month{2026, time.February}, [][2]string{
			{"2026-02-01", "2026-02-15"}, {"2026-02-16", "2026-02-28"},
		}},
		{"febrero bisiesto", Month{2024, time.February}, [][2]string{
			{"2024-02-01", "2024-02-15"}, {"2024-02-16", "2024-02-29"},
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			blocks, err := Split(nil, c.month, c.month, 15)
			if err != nil {
				t.Fatalf("Split: %v", err)
			}
			assertRanges(t, blocks, c.want)
		})
	}
}

func TestSplitOtherIntervals(t *testing.T) {
	july := Month{2025, time.July} // 31 días

	blocks, err := Split(nil, july, july, 10)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	// El sobrante de un día se absorbe en el último bloque.
	assertRanges(t, blocks, [][2]string{
		{"2025-07-01", "2025-07-10"},
		{"2025-07-11", "2025-07-20"},
		{"2025-07-21", "2025-07-31"},
	})

	blocks, err = Split(nil, july, july, 1)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if got, want := len(blocks), 31; got != want {
		t.Errorf("bloques con intervalo 1 = %d, se esperaban %d", got, want)
	}

	blocks, err = Split(nil, july, july, 60)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	assertRanges(t, blocks, [][2]string{{"2025-07-01", "2025-07-31"}})
}

func TestSplitSpansMonthRangeInclusive(t *testing.T) {
	blocks, err := Split(nil, Month{2025, time.June}, Month{2025, time.August}, 15)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if got, want := len(blocks), 6; got != want {
		t.Fatalf("bloques = %d, se esperaban %d", got, want)
	}
	if got, want := blocks[0].From.Format("2006-01-02"), "2025-06-01"; got != want {
		t.Errorf("inicio = %s, se esperaba %s", got, want)
	}
	if got, want := blocks[5].To.Format("2006-01-02"), "2025-08-31"; got != want {
		t.Errorf("fin = %s, se esperaba %s", got, want)
	}
}

func TestSplitFiltersOutsideRange(t *testing.T) {
	days := ByDay([]elitech.Reading{
		{Time: time.Date(2025, 5, 31, 12, 0, 0, 0, time.UTC), Temp: 1},
		at(3, 12, 5),
		{Time: time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC), Temp: 9},
	}, nil, nil)

	june := Month{2025, time.June}
	blocks, err := Split(days, june, june, 15)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	total := 0
	for _, b := range blocks {
		total += b.Summarize(nil, nil).Count
	}
	if got, want := total, 1; got != want {
		t.Errorf("lecturas dentro del rango = %d, se esperaba %d", got, want)
	}
}

func TestSplitFillsMissingDays(t *testing.T) {
	june := Month{2025, time.June}
	blocks, err := Split(ByDay([]elitech.Reading{at(3, 12, 5)}, nil, nil), june, june, 15)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	first := blocks[0]
	if got, want := len(first.Days), 15; got != want {
		t.Errorf("días del bloque = %d, se esperaban %d", got, want)
	}
	if got, want := first.DaysWithData(), 1; got != want {
		t.Errorf("días con datos = %d, se esperaba %d", got, want)
	}
	if got, want := first.Span(), 15; got != want {
		t.Errorf("Span() = %d, se esperaba %d", got, want)
	}
	if blocks[1].DaysWithData() != 0 {
		t.Error("la segunda quincena debería quedar sin datos")
	}
}

func TestSplitRejectsBadArguments(t *testing.T) {
	june := Month{2025, time.June}
	july := Month{2025, time.July}

	if _, err := Split(nil, june, june, 0); err == nil {
		t.Error("un intervalo de 0 días debería fallar")
	}
	if _, err := Split(nil, july, june, 15); err == nil {
		t.Error("un rango invertido debería fallar")
	}
}

func TestSummarizeWeightsByReadings(t *testing.T) {
	high, low := 8.0, 2.0
	days := ByDay([]elitech.Reading{
		at(1, 0, 1.0), // por debajo
		at(1, 6, 3.0),
		at(2, 0, 9.0), // por encima
	}, &high, &low)

	june := Month{2025, time.June}
	blocks, _ := Split(days, june, june, 15)
	s := blocks[0].Summarize(&high, &low)

	if got, want := s.Count, 3; got != want {
		t.Errorf("lecturas = %d, se esperaban %d", got, want)
	}
	// Media ponderada por lecturas, no media de medias: (1+3+9)/3.
	if math.Abs(s.Mean-13.0/3.0) > 1e-9 {
		t.Errorf("media = %v, se esperaba %v", s.Mean, 13.0/3.0)
	}
	if s.Min != 1.0 || s.Max != 9.0 {
		t.Errorf("min/max = %v/%v, se esperaban 1/9", s.Min, s.Max)
	}
	if s.AboveHigh != 1 || s.BelowLow != 1 {
		t.Errorf("excursiones = %d/%d, se esperaban 1/1", s.AboveHigh, s.BelowLow)
	}
	if !s.HasExcursion {
		t.Error("HasExcursion debería ser cierto al conocerse los límites")
	}
}

func TestSummarizeEmptyBlock(t *testing.T) {
	june := Month{2025, time.June}
	blocks, _ := Split(nil, june, june, 15)
	if got := blocks[0].Summarize(nil, nil); got.Count != 0 {
		t.Errorf("lecturas = %d, se esperaban 0", got.Count)
	}
}

func assertRanges(t *testing.T, blocks []Block, want [][2]string) {
	t.Helper()
	if len(blocks) != len(want) {
		t.Fatalf("bloques = %d, se esperaban %d", len(blocks), len(want))
	}
	for i, w := range want {
		from := blocks[i].From.Format("2006-01-02")
		to := blocks[i].To.Format("2006-01-02")
		if from != w[0] || to != w[1] {
			t.Errorf("bloque %d = %s..%s, se esperaba %s..%s", i, from, to, w[0], w[1])
		}
	}
}
