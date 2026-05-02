//go:build ignore

package main

import (
	"math"
	"testing"
)

func TestDividir(t *testing.T) {
	tests := []struct {
		a, b         float64
		esperado     float64
		debeFallar   bool
	}{
		{10, 2, 5, false},
		{7, 2, 3.5, false},
		{0, 5, 0, false},
		{-10, 2, -5, false},
		{5, 0, 0, true},   // división por cero debe devolver error
		{0, 0, 0, true},   // división por cero debe devolver error
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
				t.Errorf("Dividir(%g, %g) = %g, esperado %g", tt.a, tt.b, resultado, tt.esperado)
			}
		}
	}
}

func TestEstadisticas(t *testing.T) {
	tests := []struct {
		nums      []int
		min, max  int
		promedio  int
		esperaErr bool
	}{
		{[]int{1, 2, 3, 4, 5}, 1, 5, 3, false},
		{[]int{10}, 10, 10, 10, false},
		{[]int{-5, 0, 5}, -5, 5, 0, false},
		{[]int{7, 7, 7}, 7, 7, 7, false},
		{[]int{}, 0, 0, 0, true}, // slice vacío debe dar error
	}

	for _, tt := range tests {
		min, max, promedio, err := Estadisticas(tt.nums)
		if tt.esperaErr {
			if err == nil {
				t.Errorf("Estadisticas(%v) debería devolver error para slice vacío", tt.nums)
			}
			continue
		}
		if err != nil {
			t.Errorf("Estadisticas(%v) error inesperado: %v", tt.nums, err)
			continue
		}
		if min != tt.min {
			t.Errorf("Estadisticas(%v) min = %d, esperado %d", tt.nums, min, tt.min)
		}
		if max != tt.max {
			t.Errorf("Estadisticas(%v) max = %d, esperado %d", tt.nums, max, tt.max)
		}
		if promedio != tt.promedio {
			t.Errorf("Estadisticas(%v) promedio = %d, esperado %d", tt.nums, promedio, tt.promedio)
		}
	}
}
