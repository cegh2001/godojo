//go:build ignore

package main

import "fmt"

// Dividir divide dos números float64.
// Si el divisor (b) es 0, debe devolver un error con mensaje descriptivo.
// Pista: usá fmt.Errorf() para crear el error.
func Dividir(a, b float64) (float64, error) {
	// TODO: Validar que b no sea 0. Si es 0, devolver error.
	// TODO: Si b no es 0, devolver a / b y nil.
	return 0, fmt.Errorf("no implementado")
}

// EsMayor devuelve el mayor de dos números enteros.
// Si son iguales, devuelve cualquiera de los dos.
func EsMayor(a, b int) int {
	// TODO: Comparar a y b con if/else y devolver el mayor
	return 0
}

// SaludoPersonalizado devuelve un saludo en diferentes idiomas.
// - "es" → "¡Hola, {nombre}!"
// - "en" → "Hello, {nombre}!"
// - "fr" → "Bonjour, {nombre}!"
// Para cualquier otro idioma, usar el saludo en español.
func SaludoPersonalizado(nombre, idioma string) string {
	// TODO: Usar switch para elegir el saludo según el idioma
	// TODO: Concatenar el saludo con el nombre
	return ""
}
