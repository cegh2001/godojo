//go:build ignore

package main

import (
	"math"
	"testing"
)

func TestDividir(t *testing.T) {
	tests := []struct {
		a, b       float64
		esperado   float64
		debeFallar bool
	}{
		{10, 2, 5, false},
		{7, 2, 3.5, false},
		{0, 5, 0, false},
		{-10, 2, -5, false},
		{5, 0, 0, true}, // división por cero
		{0, 0, 0, true}, // división por cero
	}

	for _, tt := range tests {
		resultado, err := Dividir(tt.a, tt.b)
		if tt.debeFallar {
			if err == nil {
				t.Errorf("Dividir(%g, %g) debería devolver error por división por cero", tt.a, tt.b)
			}
		} else {
			if err != nil {
				t.Errorf("Dividir(%g, %g) error inesperado: %v", tt.a, tt.b, err)
			}
			if math.Abs(resultado-tt.esperado) > 0.0001 {
				t.Errorf("Dividir(%g, %g) = %g, se esperaba %g", tt.a, tt.b, resultado, tt.esperado)
			}
		}
	}
}

func TestEsMayor(t *testing.T) {
	tests := []struct {
		a, b     int
		esperado int
	}{
		{5, 3, 5},
		{3, 5, 5},
		{7, 7, 7},
		{-1, -5, -1},
		{0, -10, 0},
	}

	for _, tt := range tests {
		resultado := EsMayor(tt.a, tt.b)
		if resultado != tt.esperado {
			t.Errorf("EsMayor(%d, %d) = %d, se esperaba %d", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestSaludoPersonalizado(t *testing.T) {
	tests := []struct {
		nombre   string
		idioma   string
		esperado string
	}{
		{"Gopher", "es", "¡Hola, Gopher!"},
		{"Gopher", "en", "Hello, Gopher!"},
		{"Gopher", "fr", "Bonjour, Gopher!"},
		{"Mundo", "es", "¡Hola, Mundo!"},
		{"Mundo", "en", "Hello, Mundo!"},
		{"", "es", "¡Hola, !"},
	}

	for _, tt := range tests {
		resultado := SaludoPersonalizado(tt.nombre, tt.idioma)
		if resultado != tt.esperado {
			t.Errorf("SaludoPersonalizado(%q, %q) = %q, se esperaba %q", tt.nombre, tt.idioma, resultado, tt.esperado)
		}
	}
}
