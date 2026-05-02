//go:build ignore

package main

import (
	"reflect"
	"testing"
)

func TestContarPalabras(t *testing.T) {
	tests := []struct {
		texto    string
		esperado map[string]int
	}{
		{"hola mundo hola", map[string]int{"hola": 2, "mundo": 1}},
		{"go go go", map[string]int{"go": 3}},
		{"una sola", map[string]int{"una": 1, "sola": 1}},
		{"", map[string]int{}},
		{"Hola hola HOLA", map[string]int{"Hola": 1, "hola": 1, "HOLA": 1}},
	}

	for _, tt := range tests {
		resultado := ContarPalabras(tt.texto)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("ContarPalabras(%q) = %v, esperado %v", tt.texto, resultado, tt.esperado)
		}
	}
}

func TestFrecuenciaLetras(t *testing.T) {
	tests := []struct {
		palabra  string
		esperado map[rune]int
	}{
		{"aaa", map[rune]int{'a': 3}},
		{"abc", map[rune]int{'a': 1, 'b': 1, 'c': 1}},
		{"", map[rune]int{}},
		{"GoGo", map[rune]int{'G': 2, 'o': 2}},
		{"a b", map[rune]int{'a': 1, ' ': 1, 'b': 1}},
	}

	for _, tt := range tests {
		resultado := FrecuenciaLetras(tt.palabra)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("FrecuenciaLetras(%q) = %v, esperado %v", tt.palabra, resultado, tt.esperado)
		}
	}
}
