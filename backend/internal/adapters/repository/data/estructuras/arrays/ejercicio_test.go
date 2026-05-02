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
			t.Errorf("EncontrarMaximo(%v) = %d, esperado %d", tt.arr, resultado, tt.esperado)
		}
	}
}

func TestInvertirArray(t *testing.T) {
	tests := []struct {
		arr      [5]int
		esperado [5]int
	}{
		{[5]int{1, 2, 3, 4, 5}, [5]int{5, 4, 3, 2, 1}},
		{[5]int{1, 1, 1, 1, 1}, [5]int{1, 1, 1, 1, 1}},
		{[5]int{1, 2, 2, 2, 3}, [5]int{3, 2, 2, 2, 1}},
		{[5]int{-1, 0, 1, 0, -1}, [5]int{-1, 0, 1, 0, -1}},
	}

	for _, tt := range tests {
		resultado := InvertirArray(tt.arr)
		if resultado != tt.esperado {
			t.Errorf("InvertirArray(%v) = %v, esperado %v", tt.arr, resultado, tt.esperado)
		}
	}
}
