//go:build ignore

package main

import "math"

// Figura es una interfaz para formas geométricas.
type Figura interface {
	Area() float64
	Perimetro() float64
}

// Rectangulo representa un rectángulo con ancho y alto.
type Rectangulo struct {
	Ancho float64
	Alto  float64
}

// Area calcula el área del rectángulo.
func (r Rectangulo) Area() float64 {
	// TODO: Implementar área = ancho * alto
	return 0
}

// Perimetro calcula el perímetro del rectángulo.
func (r Rectangulo) Perimetro() float64 {
	// TODO: Implementar perímetro = 2 * (ancho + alto)
	return 0
}

// Circulo representa un círculo con un radio.
type Circulo struct {
	Radio float64
}

// Area calcula el área del círculo.
func (c Circulo) Area() float64 {
	// TODO: Implementar área = π * radio²
	_ = math.Pi // Pista: usá math.Pi
	return 0
}

// Perimetro calcula el perímetro (circunferencia) del círculo.
func (c Circulo) Perimetro() float64 {
	// TODO: Implementar perímetro = 2 * π * radio
	return 0
}

// CalcularAreaTotal suma el área de todas las figuras en el slice.
func CalcularAreaTotal(figuras []Figura) float64 {
	// TODO: Recorrer las figuras y sumar sus áreas
	return 0
}
