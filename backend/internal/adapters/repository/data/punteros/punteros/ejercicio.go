//go:build ignore

package main

// Intercambiar intercambia los valores de dos enteros usando punteros.
// Pista: usá el operador * para desreferenciar y leer/escribir el valor.
func Intercambiar(a, b *int) {
	// TODO: Guardar el valor de *a en una variable temporal
	// TODO: Asignar *b a *a
	// TODO: Asignar la variable temporal a *b
}

// Incrementar incrementa en 1 el valor apuntado por n.
func Incrementar(n *int) {
	// TODO: Incrementar *n en 1 usando el operador de desreferencia (*n)
}

// NuevoEntero crea un nuevo entero en el heap y devuelve un puntero a él.
// Pista: usá la función incorporada new() de Go, que reserva memoria y devuelve un puntero.
func NuevoEntero(valor int) *int {
	// TODO: Crear un nuevo int con new(int)
	// TODO: Asignarle el valor recibido (*p = valor)
	// TODO: Devolver el puntero
	return nil
}
