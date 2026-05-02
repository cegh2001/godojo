//go:build ignore

package main

import "testing"

func TestSaludar(t *testing.T) {
	tests := []struct {
		nombre   string
		esperado string
	}{
		{"Mundo", "¡Hola, Mundo!"},
		{"Gopher", "¡Hola, Gopher!"},
		{"", "¡Hola, !"},
		{"María", "¡Hola, María!"},
		{"Go-go", "¡Hola, Go-go!"},
	}

	for _, tt := range tests {
		resultado := Saludar(tt.nombre)
		if resultado != tt.esperado {
			t.Errorf("Saludar(%q) = %q, esperado %q", tt.nombre, resultado, tt.esperado)
		}
	}
}
