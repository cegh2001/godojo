//go:build ignore

package main

import "testing"

func TestEncontrarMaximo(t *testing.T) {
	tests := []struct {
		arr      [5]int
		esperado int
	}{
		{[5]int{1, 2, 3, 4, 5}, 5},
		{[5]int{5, 4, 3, 2, 1}, 5},
		{[5]int{-1, -2, -3, -4, -5}, -1},
		{[5]int{0, 0, 0, 0, 0}, 0},
		{[5]int{7, 3, 9, 2, 5}, 9},
	}

	for _, tt := range tests {
		resultado := EncontrarMaximo(tt.arr)
		if resultado != tt.esperado {
			t.Errorf("EncontrarMaximo(%v) = %d, se esperaba %d", tt.arr, resultado, tt.esperado)
		}
	}
}

func TestSumarArray(t *testing.T) {
	tests := []struct {
		arr      [5]int
		esperado int
	}{
		{[5]int{1, 2, 3, 4, 5}, 15},
		{[5]int{0, 0, 0, 0, 0}, 0},
		{[5]int{-1, 1, -1, 1, 0}, 0},
		{[5]int{10, 20, 30, 40, 50}, 150},
	}

	for _, tt := range tests {
		resultado := SumarArray(tt.arr)
		if resultado != tt.esperado {
			t.Errorf("SumarArray(%v) = %d, se esperaba %d", tt.arr, resultado, tt.esperado)
		}
	}
}

func TestContiene(t *testing.T) {
	tests := []struct {
		arr      [5]int
		valor    int
		esperado bool
	}{
		{[5]int{1, 2, 3, 4, 5}, 3, true},
		{[5]int{1, 2, 3, 4, 5}, 6, false},
		{[5]int{0, 0, 0, 0, 0}, 0, true},
		{[5]int{1, 2, 3, 4, 5}, 0, false},
		{[5]int{-1, -2, -3, 0, 1}, -2, true},
	}

	for _, tt := range tests {
		resultado := Contiene(tt.arr, tt.valor)
		if resultado != tt.esperado {
			t.Errorf("Contiene(%v, %d) = %v, se esperaba %v", tt.arr, tt.valor, resultado, tt.esperado)
		}
	}
}
