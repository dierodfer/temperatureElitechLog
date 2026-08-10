// Package stats agrega las lecturas por día y las agrupa en bloques de fechas.
package stats

import (
	"sort"
	"time"

	"github.com/dierodfer/temperatureElitechLog/internal/elitech"
)

// Day resume todas las lecturas de una misma fecha.
type Day struct {
	Date  time.Time // medianoche del día, sin componente horaria
	Count int
	Mean  float64
	Min   float64
	Max   float64
	MinAt time.Time // instante exacto del mínimo
	MaxAt time.Time // instante exacto del máximo
	Above int       // lecturas por encima del límite de alarma superior
	Below int       // lecturas por debajo del límite de alarma inferior
}

// ByDay agrupa las lecturas por fecha natural y devuelve los días ordenados
// cronológicamente. Los días sin lecturas no aparecen: los rellena quien monta
// los bloques, que es quien sabe qué rango debe cubrirse.
//
// high y low son los límites de alarma del registrador; si son nil, no se
// contabilizan excursiones.
func ByDay(readings []elitech.Reading, high, low *float64) []Day {
	type acc struct {
		sum          float64
		count        int
		min, max     float64
		minAt, maxAt time.Time
		above, below int
	}

	groups := map[time.Time]*acc{}
	for _, r := range readings {
		key := dayOf(r.Time)
		a := groups[key]
		if a == nil {
			a = &acc{min: r.Temp, max: r.Temp, minAt: r.Time, maxAt: r.Time}
			groups[key] = a
		}
		a.sum += r.Temp
		a.count++
		if r.Temp < a.min {
			a.min, a.minAt = r.Temp, r.Time
		}
		if r.Temp > a.max {
			a.max, a.maxAt = r.Temp, r.Time
		}
		if high != nil && r.Temp > *high {
			a.above++
		}
		if low != nil && r.Temp < *low {
			a.below++
		}
	}

	days := make([]Day, 0, len(groups))
	for date, a := range groups {
		days = append(days, Day{
			Date:  date,
			Count: a.count,
			Mean:  a.sum / float64(a.count),
			Min:   a.min,
			Max:   a.max,
			MinAt: a.minAt,
			MaxAt: a.maxAt,
			Above: a.above,
			Below: a.below,
		})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date.Before(days[j].Date) })
	return days
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
