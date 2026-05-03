//go:build ignore

package main

// Contador representa un contador numérico simple.
type Contador struct {
	valor int
}

// Incrementar aumenta el valor del Contador en 1.
// Este es un método con receiver de tipo PUNTERO (*Contador).
// Los métodos con pointer receiver pueden modificar el struct original.
func (c *Contador) Incrementar() {
	// TODO: Incrementar c.valor en 1
}

// Valor devuelve el valor actual del Contador.
// Este es un método con receiver de tipo VALOR (Contador).
// Los métodos con value receiver NO pueden modificar el struct original.
func (c Contador) Valor() int {
	// TODO: Devolver c.valor
	return 0
}

// NuevoContador es un constructor que crea un Contador y devuelve un puntero.
// Los constructores que devuelven punteros son comunes en Go cuando
// el struct se va a modificar mediante métodos con pointer receiver.
func NuevoContador(inicial int) *Contador {
	// TODO: Crear un Contador con valor 'inicial' y devolver su dirección con &
	return nil
}
