//go:build ignore

package main

import "testing"

func TestConvertirAMayusculas(t *testing.T) {
	tests := []struct {
		entrada  string
		esperado string
	}{
		{"hola", "HOLA"},
		{"Go", "GO"},
		{"Gopher", "GOPHER"},
		{"", ""},
		{"aBc", "ABC"},
	}

	for _, tt := range tests {
		resultado := ConvertirAMayusculas(tt.entrada)
		if resultado != tt.esperado {
			t.Errorf("ConvertirAMayusculas(%q) = %q, se esperaba %q", tt.entrada, resultado, tt.esperado)
		}
	}
}

func TestEsPositivo(t *testing.T) {
	tests := []struct {
		n        int
		esperado bool
	}{
		{5, true},
		{1, true},
		{0, false},
		{-1, false},
		{-100, false},
	}

	for _, tt := range tests {
		resultado := EsPositivo(tt.n)
		if resultado != tt.esperado {
			t.Errorf("EsPositivo(%d) = %v, se esperaba %v", tt.n, resultado, tt.esperado)
		}
	}
}

func TestRedondear(t *testing.T) {
	tests := []struct {
		f        float64
		esperado int
	}{
		{3.4, 3},
		{3.5, 4},
		{3.6, 4},
		{0.0, 0},
		{0.2, 0},
		{0.5, 1},
		{0.9, 1},
	}

	for _, tt := range tests {
		resultado := Redondear(tt.f)
		if resultado != tt.esperado {
			t.Errorf("Redondear(%g) = %d, se esperaba %d", tt.f, resultado, tt.esperado)
		}
	}
}

func TestLongitudDeString(t *testing.T) {
	tests := []struct {
		s        string
		esperado int
	}{
		{"hola", 4},
		{"Go", 2},
		{"", 0},
		{"🚀", 4}, // los emojis ocupan más de un byte
	}

	for _, tt := range tests {
		resultado := LongitudDeString(tt.s)
		if resultado != tt.esperado {
			t.Errorf("LongitudDeString(%q) = %d, se esperaba %d", tt.s, resultado, tt.esperado)
		}
	}
}
