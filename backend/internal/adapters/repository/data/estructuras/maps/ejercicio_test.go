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
			t.Errorf("ContarPalabras(%q) = %v, se esperaba %v", tt.texto, resultado, tt.esperado)
		}
	}
}

func TestExisteClave(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	tests := []struct {
		clave    string
		esperado bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"", false},
	}

	for _, tt := range tests {
		resultado := ExisteClave(m, tt.clave)
		if resultado != tt.esperado {
			t.Errorf("ExisteClave(m, %q) = %v, se esperaba %v", tt.clave, resultado, tt.esperado)
		}
	}
}

func TestExisteClave_MapaVacio(t *testing.T) {
	m := map[string]int{}
	resultado := ExisteClave(m, "x")
	if resultado != false {
		t.Errorf("ExisteClave({}, %q) = %v, se esperaba false", "x", resultado)
	}
}
