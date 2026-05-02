//go:build ignore

package main

import "testing"

func TestClasificarNumero(t *testing.T) {
	tests := []struct {
		n        int
		esperado string
	}{
		{5, "positivo"},
		{1, "positivo"},
		{0, "cero"},
		{-1, "negativo"},
		{-100, "negativo"},
	}

	for _, tt := range tests {
		resultado := ClasificarNumero(tt.n)
		if resultado != tt.esperado {
			t.Errorf("ClasificarNumero(%d) = %q, se esperaba %q", tt.n, resultado, tt.esperado)
		}
	}
}

func TestSumarHasta(t *testing.T) {
	tests := []struct {
		n        int
		esperado int
	}{
		{1, 1},
		{2, 3},
		{3, 6},
		{5, 15},
		{10, 55},
		{0, 0},
	}

	for _, tt := range tests {
		resultado := SumarHasta(tt.n)
		if resultado != tt.esperado {
			t.Errorf("SumarHasta(%d) = %d, se esperaba %d", tt.n, resultado, tt.esperado)
		}
	}
}

func TestDiaDeLaSemana(t *testing.T) {
	tests := []struct {
		n        int
		esperado string
	}{
		{1, "Lunes"},
		{2, "Martes"},
		{3, "Miércoles"},
		{4, "Jueves"},
		{5, "Viernes"},
		{6, "Sábado"},
		{7, "Domingo"},
		{0, "Número inválido"},
		{8, "Número inválido"},
	}

	for _, tt := range tests {
		resultado := DiaDeLaSemana(tt.n)
		if resultado != tt.esperado {
			t.Errorf("DiaDeLaSemana(%d) = %q, se esperaba %q", tt.n, resultado, tt.esperado)
		}
	}
}
