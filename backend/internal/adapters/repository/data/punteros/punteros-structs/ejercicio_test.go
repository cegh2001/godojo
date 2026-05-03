//go:build ignore

package main

import "testing"

func TestContador_Valor(t *testing.T) {
	tests := []struct {
		inicial  int
		esperado int
	}{
		{0, 0},
		{5, 5},
		{100, 100},
		{-10, -10},
	}

	for _, tt := range tests {
		c := NuevoContador(tt.inicial)
		resultado := c.Valor()
		if resultado != tt.esperado {
			t.Errorf("NuevoContador(%d).Valor() = %d, se esperaba %d", tt.inicial, resultado, tt.esperado)
		}
	}
}

func TestContador_Incrementar(t *testing.T) {
	tests := []struct {
		inicial          int
		veces            int
		esperadoFinal    int
	}{
		{0, 1, 1},
		{5, 3, 8},
		{10, 10, 20},
		{0, 0, 0},
	}

	for _, tt := range tests {
		c := NuevoContador(tt.inicial)
		for i := 0; i < tt.veces; i++ {
			c.Incrementar()
		}
		resultado := c.Valor()
		if resultado != tt.esperadoFinal {
			t.Errorf("NuevoContador(%d).Incrementar() ×%d veces → Valor() = %d, se esperaba %d",
				tt.inicial, tt.veces, resultado, tt.esperadoFinal)
		}
	}
}

func TestContador_PointerVsValueReceiver(t *testing.T) {
	// Verificar que Incrementar (pointer receiver) modifica el struct original
	c := NuevoContador(10)
	c.Incrementar()
	if c.Valor() != 11 {
		t.Errorf("Después de Incrementar(), Valor() debería ser 11, pero es %d", c.Valor())
	}

	// Verificar que Valor() (value receiver) no modifica nada
	v := c.Valor()
	_ = v // solo lectura
	if c.Valor() != 11 {
		t.Errorf("Valor() no debería modificar el Contador, pero cambió a %d", c.Valor())
	}
}

func TestNuevoContador_DevuelvePuntero(t *testing.T) {
	ptr := NuevoContador(0)
	if ptr == nil {
		t.Error("NuevoContador(0) devolvió nil, se esperaba un puntero válido")
	}
}
