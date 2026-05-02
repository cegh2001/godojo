//go:build ignore

package main

import (
	"math"
	"testing"
)

func TestRectangulo_Area(t *testing.T) {
	r := Rectangulo{Ancho: 5, Alto: 3}
	area := r.Area()
	if math.Abs(area-15.0) > 0.0001 {
		t.Errorf("Area del rectángulo 5x3 = %g, esperado 15", area)
	}
}

func TestRectangulo_Perimetro(t *testing.T) {
	r := Rectangulo{Ancho: 5, Alto: 3}
	perimetro := r.Perimetro()
	if math.Abs(perimetro-16.0) > 0.0001 {
		t.Errorf("Perímetro del rectángulo 5x3 = %g, esperado 16", perimetro)
	}
}

func TestCirculo_Area(t *testing.T) {
	c := Circulo{Radio: 2}
	area := c.Area()
	esperado := math.Pi * 4
	if math.Abs(area-esperado) > 0.0001 {
		t.Errorf("Área del círculo r=2 = %g, esperado %g", area, esperado)
	}
}

func TestCirculo_Perimetro(t *testing.T) {
	c := Circulo{Radio: 2}
	perimetro := c.Perimetro()
	esperado := 2 * math.Pi * 2
	if math.Abs(perimetro-esperado) > 0.0001 {
		t.Errorf("Perímetro del círculo r=2 = %g, esperado %g", perimetro, esperado)
	}
}

func TestCalcularAreaTotal(t *testing.T) {
	figuras := []Figura{
		Rectangulo{Ancho: 2, Alto: 3},  // área = 6
		Circulo{Radio: 1},               // área = π
	}
	areaTotal := CalcularAreaTotal(figuras)
	esperado := 6.0 + math.Pi
	if math.Abs(areaTotal-esperado) > 0.0001 {
		t.Errorf("CalcularAreaTotal = %g, esperado %g", areaTotal, esperado)
	}
}

func TestCalcularAreaTotal_Vacio(t *testing.T) {
	figuras := []Figura{}
	areaTotal := CalcularAreaTotal(figuras)
	if areaTotal != 0 {
		t.Errorf("CalcularAreaTotal vacío = %g, esperado 0", areaTotal)
	}
}

func TestRectanguloEsFigura(t *testing.T) {
	var f Figura = Rectangulo{Ancho: 1, Alto: 1}
	_ = f // Verifica que Rectangulo implementa la interfaz Figura
}

func TestCirculoEsFigura(t *testing.T) {
	var f Figura = Circulo{Radio: 1}
	_ = f // Verifica que Circulo implementa la interfaz Figura
}
