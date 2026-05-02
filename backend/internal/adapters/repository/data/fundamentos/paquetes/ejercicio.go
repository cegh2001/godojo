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
	fmt.Println("Ejercicio de paquetes — importá y usá el paquete calculadora")
	fmt.Println("Resultado:", UsarSuma(5, 3))
}
