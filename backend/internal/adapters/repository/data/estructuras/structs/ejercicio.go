//go:build ignore

package main

// Rectangulo representa un rectángulo con ancho y alto.
type Rectangulo struct {
	Ancho float64
	Alto  float64
}

// Area calcula el área del rectángulo (ancho × alto).
// Este es un método de Rectangulo — notá el "receiver" (r Rectangulo) antes del nombre.
func (r Rectangulo) Area() float64 {
	// TODO: Devolver r.Ancho * r.Alto
	return 0
}

// Perimetro calcula el perímetro del rectángulo (2 × (ancho + alto)).
func (r Rectangulo) Perimetro() float64 {
	// TODO: Devolver 2 * (r.Ancho + r.Alto)
	return 0
}

// EsCuadrado determina si un rectángulo es un cuadrado (ancho == alto).
// Esta es una función normal, no un método.
func EsCuadrado(r Rectangulo) bool {
	// TODO: Comparar r.Ancho y r.Alto
	return false
}

// NuevoRectangulo es un constructor que crea y devuelve un Rectangulo.
// Los constructores en Go suelen llamarse NuevoXxx o NewXxx.
func NuevoRectangulo(ancho, alto float64) Rectangulo {
	// TODO: Crear y devolver un Rectangulo con los valores recibidos
	return Rectangulo{}
}
