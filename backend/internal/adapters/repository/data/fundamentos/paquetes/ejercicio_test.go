//go:build ignore

package main

import (
	"testing"
)

func TestUsarPaqueteCalculadora(t *testing.T) {
	// Verificamos que podemos llamar a la función exportada Sumar
	resultado := UsarSuma(5, 3)
	if resultado != 8 {
		t.Errorf("UsarSuma(5, 3) = %d, esperado 8", resultado)
	}

	resultado = UsarSuma(-1, 1)
	if resultado != 0 {
		t.Errorf("UsarSuma(-1, 1) = %d, esperado 0", resultado)
	}
}

func TestCalculadoraValidar_EsVisible(t *testing.T) {
	// Verificamos que la función Sumar funciona correctamente
	resultado := UsarSuma(10, 20)
	if resultado != 30 {
		t.Errorf("UsarSuma(10, 20) = %d, esperado 30", resultado)
	}
}
