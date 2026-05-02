//go:build ignore

package main

import "testing"

func TestSumarEnteros(t *testing.T) {
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
		resultado := SumarEnteros(tt.a, tt.b)
		if resultado != tt.esperado {
			t.Errorf("SumarEnteros(%d, %d) = %d, esperado %d", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestConcatenar(t *testing.T) {
	tests := []struct {
		a, b     string
		esperado string
	}{
		{"Hola", "Mundo", "HolaMundo"},
		{"", "", ""},
		{"Go", "", "Go"},
		{"", "Go", "Go"},
		{"🚀", "Go", "🚀Go"},
	}

	for _, tt := range tests {
		resultado := Concatenar(tt.a, tt.b)
		if resultado != tt.esperado {
			t.Errorf("Concatenar(%q, %q) = %q, esperado %q", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestEsMayorDeEdad(t *testing.T) {
	tests := []struct {
		edad     int
		esperado bool
	}{
		{18, true},
		{21, true},
		{17, false},
		{0, false},
		{-1, false},
		{100, true},
	}

	for _, tt := range tests {
		resultado := EsMayorDeEdad(tt.edad)
		if resultado != tt.esperado {
			t.Errorf("EsMayorDeEdad(%d) = %v, esperado %v", tt.edad, resultado, tt.esperado)
		}
	}
}
