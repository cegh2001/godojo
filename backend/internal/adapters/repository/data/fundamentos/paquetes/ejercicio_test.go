//go:build ignore

package main

import "testing"

func TestUsarSuma(t *testing.T) {
	tests := []struct {
		a, b     int
		esperado int
	}{
		{5, 3, 8},
		{10, 20, 30},
		{-1, 1, 0},
		{0, 0, 0},
		{-5, -3, -8},
	}

	for _, tt := range tests {
		resultado := UsarSuma(tt.a, tt.b)
		if resultado != tt.esperado {
			t.Errorf("UsarSuma(%d, %d) = %d, se esperaba %d", tt.a, tt.b, resultado, tt.esperado)
		}
	}
}

func TestUsarSuma_LlamaAlPaquete(t *testing.T) {
	// Verifica que la función UsarSuma efectivamente llama al paquete calculadora.
	// Si el paquete no está implementado correctamente, este test falla.
	resultado := UsarSuma(100, 50)
	if resultado != 150 {
		t.Errorf("UsarSuma(100, 50) = %d, se esperaba 150. ¿Importaste y usaste el paquete calculadora?", resultado)
	}
}
