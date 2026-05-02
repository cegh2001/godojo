//go:build ignore

package main

import "strings"

// ContarPalabras cuenta la frecuencia de cada palabra en un texto.
// Las palabras se separan por espacios. Es sensible a mayúsculas/minúsculas.
// Si el texto está vacío, devuelve un map vacío (no nil).
// Pista: usá strings.Fields() para dividir el texto en palabras.
func ContarPalabras(texto string) map[string]int {
	// TODO: Crear un map vacío con make(map[string]int)
	// TODO: Para cada palabra obtenida con strings.Fields, incrementar el contador
	_ = strings.Fields // pista
	return nil
}

// ExisteClave verifica si una clave existe en un map.
// Retorna true si la clave está presente, false en caso contrario.
// Pista: usá el "comma-ok idiom": valor, ok := m[clave]
func ExisteClave(m map[string]int, clave string) bool {
	// TODO: Usar el comma-ok idiom para verificar si la clave existe
	return false
}
