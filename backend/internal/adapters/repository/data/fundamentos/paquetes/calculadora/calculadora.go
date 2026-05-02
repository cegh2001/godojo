//go:build ignore

package calculadora

// Sumar retorna la suma de dos números enteros.
// Esta función es exportada (visible desde otros paquetes).
func Sumar(a, b int) int {
	// TODO: Implementar la suma de a y b
	return 0
}

// validar verifica que el número sea no negativo.
// Esta función NO es exportada (solo visible dentro del paquete calculadora).
func validar(n int) bool {
	// TODO: Implementar validación (retornar true si n >= 0)
	return false
}
