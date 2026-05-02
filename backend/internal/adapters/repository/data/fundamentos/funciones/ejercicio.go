//go:build ignore

package main

import "fmt"

// Dividir divide dos números float64.
// Si el divisor es 0, debe devolver un error indicando "división por cero".
func Dividir(a, b float64) (float64, error) {
	// TODO: Implementar la división con manejo de error para b == 0
	return 0, fmt.Errorf("no implementado")
}

// Estadisticas calcula el mínimo, máximo y promedio de un slice de enteros.
// Si el slice está vacío, devuelve un error.
func Estadisticas(numeros []int) (min, max, promedio int, err error) {
	// TODO: Implementar el cálculo de estadísticas
	return 0, 0, 0, fmt.Errorf("no implementado")
}
