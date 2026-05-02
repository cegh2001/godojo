//go:build ignore

package main

import (
	"reflect"
	"testing"
)

func TestFiltrarPares(t *testing.T) {
	tests := []struct {
		nums     []int
		esperado []int
	}{
		{[]int{1, 2, 3, 4, 5, 6}, []int{2, 4, 6}},
		{[]int{1, 3, 5}, []int{}},
		{[]int{2, 4, 6, 8}, []int{2, 4, 6, 8}},
		{[]int{}, []int{}},
		{[]int{-2, -1, 0, 1, 2}, []int{-2, 0, 2}},
	}

	for _, tt := range tests {
		resultado := FiltrarPares(tt.nums)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("FiltrarPares(%v) = %v, esperado %v", tt.nums, resultado, tt.esperado)
		}
	}
}

func TestEliminarDuplicados(t *testing.T) {
	tests := []struct {
		nums     []int
		esperado []int
	}{
		{[]int{1, 2, 2, 3, 4, 4, 5}, []int{1, 2, 3, 4, 5}},
		{[]int{1, 1, 1, 1}, []int{1}},
		{[]int{1, 2, 3}, []int{1, 2, 3}},
		{[]int{}, []int{}},
		{[]int{5, 5, 5, 1, 1}, []int{5, 1}},
	}

	for _, tt := range tests {
		resultado := EliminarDuplicados(tt.nums)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("EliminarDuplicados(%v) = %v, esperado %v", tt.nums, resultado, tt.esperado)
		}
	}
}

func TestConcatenarSlices(t *testing.T) {
	tests := []struct {
		a, b     []int
		esperado []int
	}{
		{[]int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},
		{[]int{}, []int{1, 2}, []int{1, 2}},
		{[]int{1, 2}, []int{}, []int{1, 2}},
		{[]int{}, []int{}, []int{}},
	}

	for _, tt := range tests {
		resultado := ConcatenarSlices(tt.a, tt.b)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("ConcatenarSlices(%v, %v) = %v, esperado %v", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}
