//go:build ignore

package main

import "strings"

// ConvertirAMayusculas convierte un string a mayúsculas.
// Pista: usá strings.ToUpper() del paquete "strings".
func ConvertirAMayusculas(s string) string {
	// TODO: Usar strings.ToUpper(s)
	_ = strings.ToUpper // pista: esta función existe
	return ""
}

// EsPositivo retorna true si el número es mayor a 0.
// El 0 no se considera positivo.
func EsPositivo(n int) bool {
	// TODO: Verificar si n > 0
	return false
}

// Redondear convierte un float64 a int redondeando al entero más cercano.
// Pista: si le sumás 0.5 al número y lo convertís a int, obtenés el redondeo.
func Redondear(f float64) int {
	// TODO: Devolver int(f + 0.5)
	return 0
}

// LongitudDeString retorna la cantidad de bytes del string.
// Pista: usá la función built-in len().
func LongitudDeString(s string) int {
	// TODO: Devolver len(s)
	return 0
}
