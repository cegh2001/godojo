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
		{[]int{1, 3, 5}, nil},
		{[]int{2, 4, 6, 8}, []int{2, 4, 6, 8}},
		{[]int{}, nil},
		{[]int{-2, -1, 0, 1, 2}, []int{-2, 0, 2}},
	}

	for _, tt := range tests {
		resultado := FiltrarPares(tt.nums)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("FiltrarPares(%v) = %v, se esperaba %v", tt.nums, resultado, tt.esperado)
		}
	}
}

func TestConcatenar(t *testing.T) {
	tests := []struct {
		a, b     []int
		esperado []int
	}{
		{[]int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},
		{[]int{}, []int{1, 2}, []int{1, 2}},
		{[]int{1, 2}, []int{}, []int{1, 2}},
		{[]int{}, []int{}, nil},
	}

	for _, tt := range tests {
		resultado := Concatenar(tt.a, tt.b)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("Concatenar(%v, %v) = %v, se esperaba %v", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestContieneSlice(t *testing.T) {
	tests := []struct {
		nums     []int
		valor    int
		esperado bool
	}{
		{[]int{1, 2, 3, 4, 5}, 3, true},
		{[]int{1, 2, 3, 4, 5}, 6, false},
		{[]int{1, 2, 3}, 1, true},
		{[]int{}, 1, false},
		{[]int{-1, -2, 0, 1}, 0, true},
	}

	for _, tt := range tests {
		resultado := ContieneSlice(tt.nums, tt.valor)
		if resultado != tt.esperado {
			t.Errorf("ContieneSlice(%v, %d) = %v, se esperaba %v", tt.nums, tt.valor, resultado, tt.esperado)
		}
	}
}
