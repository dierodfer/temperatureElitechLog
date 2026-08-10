package stats

import (
	"fmt"
	"strings"
	"time"
)

// Month identifica un mes concreto del calendario.
type Month struct {
	Year  int
	Month time.Month
}

// ParseMonth acepta un mes en formato AAAA-MM, tolerando también AAAA/MM y
// MM-AAAA, que son las formas en que la gente suele teclearlo.
func ParseMonth(s string) (Month, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "/", "-"))
	if s == "" {
		return Month{}, fmt.Errorf("mes vacío")
	}
	for _, layout := range []string{"2006-01", "01-2006", "2006-1", "1-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return Month{Year: t.Year(), Month: t.Month()}, nil
		}
	}
	return Month{}, fmt.Errorf("mes %q no válido: usa el formato AAAA-MM, por ejemplo 2025-06", s)
}

// MonthOf devuelve el mes al que pertenece una fecha.
func MonthOf(t time.Time) Month { return Month{Year: t.Year(), Month: t.Month()} }

// Start es el primer día del mes.
func (m Month) Start() time.Time {
	return time.Date(m.Year, m.Month, 1, 0, 0, 0, 0, time.UTC)
}

// End es el último día del mes.
func (m Month) End() time.Time {
	return m.Start().AddDate(0, 1, -1)
}

// Before indica si el mes es anterior a otro.
func (m Month) Before(o Month) bool {
	if m.Year != o.Year {
		return m.Year < o.Year
	}
	return m.Month < o.Month
}

func (m Month) String() string { return fmt.Sprintf("%04d-%02d", m.Year, m.Month) }

var monthNames = [...]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

// Human devuelve el mes en castellano, para las cabeceras del informe.
func (m Month) Human() string {
	return fmt.Sprintf("%s %d", monthNames[int(m.Month)-1], m.Year)
}

// Block es un intervalo de días consecutivos con sus estadísticas.
type Block struct {
	From time.Time
	To   time.Time
	Days []Day // un elemento por día del intervalo; Count == 0 si no hubo lecturas
}

// Span es el número de días naturales que abarca el bloque.
func (b Block) Span() int { return int(b.To.Sub(b.From).Hours()/24) + 1 }

// DaysWithData cuenta los días del bloque que sí tienen lecturas.
func (b Block) DaysWithData() int {
	n := 0
	for _, d := range b.Days {
		if d.Count > 0 {
			n++
		}
	}
	return n
}

// Summary son los agregados de un bloque completo.
type Summary struct {
	Count        int
	Mean         float64
	Min          float64
	Max          float64
	MinAt        time.Time
	MaxAt        time.Time
	AboveHigh    int
	BelowLow     int
	HasExcursion bool // solo cierto si se conocían los límites de alarma
}

// Summarize agrega el bloque entero. Los recuentos de excursión solo se
// calculan si se pasan los límites de alarma leídos del fichero.
func (b Block) Summarize(high, low *float64) Summary {
	var s Summary
	first := true
	for _, d := range b.Days {
		if d.Count == 0 {
			continue
		}
		s.Count += d.Count
		s.Mean += d.Mean * float64(d.Count)
		s.AboveHigh += d.Above
		s.BelowLow += d.Below
		if first || d.Min < s.Min {
			s.Min, s.MinAt = d.Min, d.MinAt
		}
		if first || d.Max > s.Max {
			s.Max, s.MaxAt = d.Max, d.MaxAt
		}
		first = false
	}
	if s.Count > 0 {
		s.Mean /= float64(s.Count)
	}
	s.HasExcursion = high != nil || low != nil
	return s
}

// Split filtra los días al rango [desde, hasta] —ambos meses incluidos— y los
// reparte en bloques de interval días.
//
// Los bloques se recortan por meses naturales: dentro de cada mes se cuenta
// desde el día 1 y ningún bloque cruza jamás un cambio de mes. El sobrante
// final se absorbe en el bloque anterior cuando no llega a medio intervalo,
// que es lo que evita que un mes de 31 días termine con un bloque suelto de un
// solo día. Con el valor por defecto de 15 días salen las quincenas 1-15 y
// 16-fin de mes.
func Split(days []Day, from, to Month, interval int) ([]Block, error) {
	if interval < 1 {
		return nil, fmt.Errorf("el intervalo debe ser de al menos 1 día")
	}
	if to.Before(from) {
		return nil, fmt.Errorf("el mes inicial (%s) es posterior al final (%s)", from, to)
	}

	byDate := make(map[time.Time]Day, len(days))
	for _, d := range days {
		byDate[d.Date] = d
	}

	var blocks []Block
	for m := from; !to.Before(m); m = m.next() {
		monthEnd := m.End()
		for _, r := range monthRanges(m.Start(), monthEnd, interval) {
			start, end := r[0], r[1]

			b := Block{From: start, To: end}
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				if day, ok := byDate[d]; ok {
					b.Days = append(b.Days, day)
					continue
				}
				b.Days = append(b.Days, Day{Date: d})
			}
			blocks = append(blocks, b)
		}
	}
	return blocks, nil
}

// monthRanges trocea un mes en intervalos de interval días. Devuelve pares
// [inicio, fin]; el último se alarga hasta fin de mes si el sobrante no llega a
// medio intervalo.
func monthRanges(monthStart, monthEnd time.Time, interval int) [][2]time.Time {
	var out [][2]time.Time
	for start := monthStart; !start.After(monthEnd); start = start.AddDate(0, 0, interval) {
		end := start.AddDate(0, 0, interval-1)
		if end.After(monthEnd) {
			end = monthEnd
		}
		out = append(out, [2]time.Time{start, end})
	}

	if n := len(out); n > 1 {
		last := out[n-1]
		remainder := int(last[1].Sub(last[0]).Hours()/24) + 1
		if remainder*2 < interval {
			out[n-2][1] = last[1]
			out = out[:n-1]
		}
	}
	return out
}

func (m Month) next() Month {
	t := m.Start().AddDate(0, 1, 0)
	return Month{Year: t.Year(), Month: t.Month()}
}
