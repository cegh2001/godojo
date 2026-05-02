//go:build ignore

package main

import (
	"math"
	"testing"
)

func TestRectangulo_Area(t *testing.T) {
	tests := []struct {
		ancho, alto float64
		esperado    float64
	}{
		{5, 3, 15},
		{2, 2, 4},
		{1, 10, 10},
		{0, 5, 0},
		{2.5, 4, 10},
	}

	for _, tt := range tests {
		r := Rectangulo{Ancho: tt.ancho, Alto: tt.alto}
		resultado := r.Area()
		if math.Abs(resultado-tt.esperado) > 0.0001 {
			t.Errorf("Area() de rectángulo %gx%g = %g, se esperaba %g", tt.ancho, tt.alto, resultado, tt.esperado)
		}
	}
}

func TestRectangulo_Perimetro(t *testing.T) {
	tests := []struct {
		ancho, alto float64
		esperado    float64
	}{
		{5, 3, 16},
		{2, 2, 8},
		{1, 10, 22},
		{0, 0, 0},
		{2.5, 4, 13},
	}

	for _, tt := range tests {
		r := Rectangulo{Ancho: tt.ancho, Alto: tt.alto}
		resultado := r.Perimetro()
		if math.Abs(resultado-tt.esperado) > 0.0001 {
			t.Errorf("Perimetro() de rectángulo %gx%g = %g, se esperaba %g", tt.ancho, tt.alto, resultado, tt.esperado)
		}
	}
}

func TestEsCuadrado(t *testing.T) {
	tests := []struct {
		ancho, alto float64
		esperado    bool
	}{
		{5, 5, true},
		{3, 3, true},
		{5, 3, false},
		{3, 5, false},
		{0, 0, true},
		{2.5, 2.5, true},
	}

	for _, tt := range tests {
		r := Rectangulo{Ancho: tt.ancho, Alto: tt.alto}
		resultado := EsCuadrado(r)
		if resultado != tt.esperado {
			t.Errorf("EsCuadrado(%gx%g) = %v, se esperaba %v", tt.ancho, tt.alto, resultado, tt.esperado)
		}
	}
}

func TestNuevoRectangulo(t *testing.T) {
	r := NuevoRectangulo(4, 6)
	if r.Ancho != 4 {
		t.Errorf("NuevoRectangulo(4, 6).Ancho = %g, se esperaba 4", r.Ancho)
	}
	if r.Alto != 6 {
		t.Errorf("NuevoRectangulo(4, 6).Alto = %g, se esperaba 6", r.Alto)
	}
}
