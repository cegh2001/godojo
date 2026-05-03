//go:build ignore

package main

import (
	"strings"
	"testing"
	"time"
)

func TestFechaActual(t *testing.T) {
	resultado := FechaActual()
	if resultado == "" {
		t.Error("FechaActual() devolvió string vacío")
	}

	// Debe tener formato YYYY-MM-DD (10 caracteres)
	if len(resultado) != 10 {
		t.Errorf("FechaActual() = %q, longitud %d — se esperaba 10 caracteres (YYYY-MM-DD)",
			resultado, len(resultado))
	}

	// Verificar que contiene los guiones del formato
	if !strings.Contains(resultado, "-") {
		t.Errorf("FechaActual() = %q — formato incorrecto, debería contener guiones", resultado)
	}
}

func TestHaceCuanto(t *testing.T) {
	// Una fecha en el pasado (1 hora atrás)
	pasado := time.Now().Add(-1 * time.Hour)
	duracion := HaceCuanto(pasado)

	// Debería ser aproximadamente 1 hora (con tolerancia)
	if duracion < 59*time.Minute || duracion > 61*time.Minute {
		t.Errorf("HaceCuanto(1 hora atrás) = %v, se esperaba ~1h", duracion)
	}

	// 1 segundo atrás
	pasadoCorto := time.Now().Add(-1 * time.Second)
	duracionCorta := HaceCuanto(pasadoCorto)
	if duracionCorta <= 0 {
		t.Errorf("HaceCuanto(1 segundo atrás) = %v, debería ser positivo", duracionCorta)
	}
}

func TestEsFinde(t *testing.T) {
	// Encontrar el próximo sábado y domingo para probar
	ahora := time.Now()
	diasHastaSabado := (6 - int(ahora.Weekday()) + 7) % 7
	if diasHastaSabado == 0 {
		diasHastaSabado = 7 // evitar hoy si ya es sábado, tomar el próximo
	}
	sabado := ahora.AddDate(0, 0, diasHastaSabado)
	domingo := sabado.AddDate(0, 0, 1)
	lunes := domingo.AddDate(0, 0, 1)

	tests := []struct {
		nombre   string
		fecha    time.Time
		esperado bool
	}{
		{"sábado", sabado, true},
		{"domingo", domingo, true},
		{"lunes", lunes, false},
	}

	for _, tt := range tests {
		resultado := EsFinde(tt.fecha)
		if resultado != tt.esperado {
			t.Errorf("EsFinde(%s) = %v, se esperaba %v", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestAgregarDias(t *testing.T) {
	fechaBase := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		dias     int
		esperado string
	}{
		{0, "2024-01-01"},
		{1, "2024-01-02"},
		{30, "2024-01-31"},
		{31, "2024-02-01"},
		{-1, "2023-12-31"},
		{365, "2024-12-31"},
	}

	for _, tt := range tests {
		resultado := AgregarDias(fechaBase, tt.dias)
		formateado := resultado.Format("2006-01-02")
		if formateado != tt.esperado {
			t.Errorf("AgregarDias(2024-01-01, %d) = %s, se esperaba %s",
				tt.dias, formateado, tt.esperado)
		}
	}
}
