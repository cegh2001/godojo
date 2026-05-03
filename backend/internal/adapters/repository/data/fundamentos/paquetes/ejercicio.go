//go:build ignore

package main

import (
	"fmt"
	"paquetes/calculadora"
)

// UsarSuma utiliza el paquete calculadora para sumar dos números.
func UsarSuma(a, b int) int {
	// TODO: Llamar a la función exportada Sumar del paquete calculadora
	// Pista: calculadora.Sumar(a, b)
	return 0
}

func main() {
	fmt.Println("Ejercicio de paquetes: importa y usa el paquete calculadora")
	fmt.Println("Suma de 5 + 3:", UsarSuma(5, 3))
}
