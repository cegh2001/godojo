//go:build ignore

package main

import (
	"strings"
	"testing"
)

func TestContarHasta(t *testing.T) {
	tests := []struct {
		n         int
		nombre    string
		contiene  []string
	}{
		{
			3, "A",
			[]string{"A: 1", "A: 2", "A: 3"},
		},
		{
			1, "X",
			[]string{"X: 1"},
		},
		{
			5, "Gorutina 1",
			[]string{"Gorutina 1: 1", "Gorutina 1: 2", "Gorutina 1: 3", "Gorutina 1: 4", "Gorutina 1: 5"},
		},
	}

	for _, tt := range tests {
		resultado := ContarHasta(tt.n, tt.nombre)
		for _, esperado := range tt.contiene {
			if !strings.Contains(resultado, esperado) {
				t.Errorf("ContarHasta(%d, %q) debería contener %q, pero el resultado fue %q",
					tt.n, tt.nombre, esperado, resultado)
			}
		}
	}

	// Verificar que ContarHasta(0, "X") no tenga contenido
	resultado := ContarHasta(0, "X")
	if resultado != "" {
		t.Errorf("ContarHasta(0, \"X\") = %q, se esperaba string vacío", resultado)
	}
}

func TestEjecutarConcurrente(t *testing.T) {
	n := 3
	resultados := EjecutarConcurrente(n)

	// Debe haber 2 resultados (uno por cada gorutina)
	if len(resultados) != 2 {
		t.Fatalf("EjecutarConcurrente(%d) devolvió %d resultados, se esperaban 2", n, len(resultados))
	}

	// Cada resultado debe contener conteo de 1 a n
	for i, resultado := range resultados {
		if resultado == "" {
			t.Errorf("EjecutarConcurrente: resultado[%d] está vacío", i)
		}
		for j := 1; j <= n; j++ {
			esperado := "Gorutina"
			if !strings.Contains(resultado, esperado) {
				t.Errorf("EjecutarConcurrente: resultado[%d] debería contener %q, pero fue %q",
					i, esperado, resultado)
			}
		}
	}
}

func TestEjecutarConcurrente_DiferentesN(t *testing.T) {
	for _, n := range []int{1, 5, 10} {
		resultados := EjecutarConcurrente(n)
		if len(resultados) != 2 {
			t.Errorf("EjecutarConcurrente(%d) devolvió %d resultados, se esperaban 2", n, len(resultados))
		}
	}
}
