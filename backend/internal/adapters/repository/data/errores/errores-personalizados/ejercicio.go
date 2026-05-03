//go:build ignore

package main

import "fmt"

// ErrorDivisionCero es un tipo de error personalizado que representa
// un intento de división por cero. Implementa la interfaz error.
type ErrorDivisionCero struct{}

// Error implementa la interfaz error para ErrorDivisionCero.
// La interfaz error solo requiere un método: Error() string.
func (e ErrorDivisionCero) Error() string {
	// TODO: Devolver un mensaje descriptivo, como "no se puede dividir por cero"
	return ""
}

// DividirSeguro divide dos números float64 de forma segura.
// Si b == 0, devuelve 0 y un error de tipo ErrorDivisionCero.
// Si b != 0, devuelve el resultado de a/b y nil.
func DividirSeguro(a, b float64) (float64, error) {
	// TODO: Verificar si b == 0
	// TODO: Si es cero, devolver 0 y ErrorDivisionCero{}
	// TODO: Si no, devolver a / b y nil
	return 0, fmt.Errorf("no implementado")
}

// EsErrorDivisionCero verifica si un error es de tipo ErrorDivisionCero
// usando errors.As, incluso si el error está envuelto (wrapped).
// Pista: usá errors.As para verificar el tipo dentro de la cadena de errores.
func EsErrorDivisionCero(err error) bool {
	// TODO: Declarar una variable de tipo *ErrorDivisionCero (puntero)
	// TODO: Usar errors.As(err, &target) para verificar
	// TODO: Devolver true si errors.As tuvo éxito
	return false
}
