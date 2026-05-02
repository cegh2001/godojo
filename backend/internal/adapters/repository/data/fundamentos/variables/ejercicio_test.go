//go:build ignore

package main

import "testing"

func TestSaludar(t *testing.T) {
	tests := []struct {
		nombre   string
		esperado string
	}{
		{"Gopher", "¡Hola, Gopher!"},
		{"Mundo", "¡Hola, Mundo!"},
		{"María", "¡Hola, María!"},
	}

	for _, tt := range tests {
		resultado := Saludar(tt.nombre)
		if resultado != tt.esperado {
			t.Errorf("Saludar(%q) = %q, se esperaba %q", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestSumar(t *testing.T) {
	tests := []struct {
		a, b     int
		esperado int
	}{
		{2, 3, 5},
		{0, 0, 0},
		{-1, 1, 0},
		{-5, -3, -8},
		{100, 200, 300},
	}

	for _, tt := range tests {
		resultado := Sumar(tt.a, tt.b)
		if resultado != tt.esperado {
			t.Errorf("Sumar(%d, %d) = %d, se esperaba %d", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestEsPar(t *testing.T) {
	tests := []struct {
		n        int
		esperado bool
	}{
		{2, true},
		{4, true},
		{0, true},
		{1, false},
		{3, false},
		{-2, true},
		{-1, false},
	}

	for _, tt := range tests {
		resultado := EsPar(tt.n)
		if resultado != tt.esperado {
			t.Errorf("EsPar(%d) = %v, se esperaba %v", tt.n, resultado, tt.esperado)
		}
	}
}
