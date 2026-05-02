//go:build ignore

package main

import "strings"

// ContarPalabras cuenta la frecuencia de cada palabra en un texto.
// Las palabras se separan por espacios. Es sensible a mayúsculas/minúsculas.
func ContarPalabras(texto string) map[string]int {
	// TODO: Dividir el texto por espacios y contar cada palabra
	_ = strings.Fields // Pista: usá strings.Fields para dividir
	return nil
}

// FrecuenciaLetras cuenta la frecuencia de cada carácter (rune) en una palabra.
func FrecuenciaLetras(palabra string) map[rune]int {
	// TODO: Recorrer cada rune de la palabra y contar su frecuencia
	return nil
}
