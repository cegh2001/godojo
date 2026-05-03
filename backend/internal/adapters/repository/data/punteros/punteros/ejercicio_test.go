//go:build ignore

package main

import "testing"

func TestIntercambiar(t *testing.T) {
	tests := []struct {
		a, b        int
		espA, espB int
	}{
		{1, 2, 2, 1},
		{0, 5, 5, 0},
		{-3, 7, 7, -3},
		{10, 10, 10, 10},
		{0, 0, 0, 0},
	}

	for _, tt := range tests {
		a, b := tt.a, tt.b
		Intercambiar(&a, &b)
		if a != tt.espA || b != tt.espB {
			t.Errorf("Intercambiar(%d, %d) = (%d, %d), se esperaba (%d, %d)",
				tt.a, tt.b, a, b, tt.espA, tt.espB)
		}
	}
}

func TestIncrementar(t *testing.T) {
	tests := []struct {
		inicial  int
		esperado int
	}{
		{0, 1},
		{5, 6},
		{-1, 0},
		{999, 1000},
		{-10, -9},
	}

	for _, tt := range tests {
		n := tt.inicial
		Incrementar(&n)
		if n != tt.esperado {
			t.Errorf("Incrementar(%d) = %d, se esperaba %d", tt.inicial, n, tt.esperado)
		}
	}
}

func TestNuevoEntero(t *testing.T) {
	tests := []struct {
		valor    int
		esperado int
	}{
		{0, 0},
		{42, 42},
		{-10, -10},
		{9999, 9999},
	}

	for _, tt := range tests {
		ptr := NuevoEntero(tt.valor)
		if ptr == nil {
			t.Errorf("NuevoEntero(%d) devolvió nil, se esperaba un puntero válido", tt.valor)
			continue
		}
		if *ptr != tt.esperado {
			t.Errorf("NuevoEntero(%d) = %d, se esperaba %d", tt.valor, *ptr, tt.esperado)
		}
	}
}

func TestNuevoEntero_PunteroUnico(t *testing.T) {
	// Verificar que cada llamado a NuevoEntero devuelve un puntero distinto
	a := NuevoEntero(5)
	b := NuevoEntero(5)
	if a == b {
		t.Error("NuevoEntero debería devolver punteros diferentes en cada llamado")
	}
}
