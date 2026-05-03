//go:build ignore

package main

import "time"

// FechaActual devuelve la fecha actual en formato ISO 8601 (YYYY-MM-DD).
// Go usa una fecha de referencia específica para el formateo: "2006-01-02".
// Pista: time.Now().Format("2006-01-02")
// ¿Por qué "2006-01-02"? Es la fecha del release de Go 1.0 en formato
// americano: 01/02 03:04:05PM '06 -0700 (mes, día, hora, min, seg, año, zona).
func FechaActual() string {
	// TODO: Devolver time.Now().Format("2006-01-02")
	return ""
}

// HaceCuanto calcula cuánto tiempo pasó desde una fecha dada hasta ahora.
// Pista: usa time.Since(fecha) que devuelve un time.Duration.

// Pista: usa fecha.Weekday() y compara con time.Saturday y time.Sunday.

// Pista: usa fecha.AddDate(0, 0, dias). AddDate recibe (años, meses, días).
func AgregarDias(fecha time.Time, dias int) time.Time {
	// TODO: Devolver fecha.AddDate(0, 0, dias)
	return time.Time{}
}
