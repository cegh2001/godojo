//go:build ignore

package main

import "testing"

func TestPersona_Saludar(t *testing.T) {
	tests := []struct {
		nombre   string
		esperado string
	}{
		{"Ana", "¡Hola, soy Ana!"},
		{"Carlos", "¡Hola, soy Carlos!"},
		{"María", "¡Hola, soy María!"},
		{"", "¡Hola, soy !"},
	}

	for _, tt := range tests {
		p := Persona{Nombre: tt.nombre, Edad: 25}
		resultado := p.Saludar()
		if resultado != tt.esperado {
			t.Errorf("Persona{Nombre: %q}.Saludar() = %q, se esperaba %q", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestPersona_CumplirAños(t *testing.T) {
	tests := []struct {
		inicial  int
		veces    int
		esperado int
	}{
		{20, 1, 21},
		{0, 5, 5},
		{17, 1, 18},
		{99, 0, 99},
	}

	for _, tt := range tests {
		p := Persona{Nombre: "Test", Edad: tt.inicial}
		for i := 0; i < tt.veces; i++ {
			p.CumplirAños()
		}
		if p.Edad != tt.esperado {
			t.Errorf("CumplirAños() ×%d desde %d → Edad = %d, se esperaba %d",
				tt.veces, tt.inicial, p.Edad, tt.esperado)
		}
	}
}

func TestPersona_EsMayor(t *testing.T) {
	tests := []struct {
		edad     int
		esperado bool
	}{
		{18, true},
		{21, true},
		{65, true},
		{17, false},
		{0, false},
		{-1, false},
	}

	for _, tt := range tests {
		p := Persona{Nombre: "Test", Edad: tt.edad}
		resultado := p.EsMayor()
		if resultado != tt.esperado {
			t.Errorf("Persona{Edad: %d}.EsMayor() = %v, se esperaba %v", tt.edad, resultado, tt.esperado)
		}
	}
}

func TestPersona_SaludarNoModifica(t *testing.T) {
	p := Persona{Nombre: "Juan", Edad: 30}
	_ = p.Saludar()
	if p.Edad != 30 || p.Nombre != "Juan" {
		t.Error("Saludar() no debería modificar los campos del struct")
	}
}
