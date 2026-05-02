//go:build ignore

package main

import (
	"reflect"
	"testing"
)

func TestFizzBuzz(t *testing.T) {
	tests := []struct {
		n        int
		esperado []string
	}{
		{1, []string{"1"}},
		{3, []string{"1", "2", "Fizz"}},
		{5, []string{"1", "2", "Fizz", "4", "Buzz"}},
		{15, []string{"1", "2", "Fizz", "4", "Buzz", "Fizz", "7", "8", "Fizz", "Buzz", "11", "Fizz", "13", "14", "FizzBuzz"}},
		{0, []string{}},
	}

	for _, tt := range tests {
		resultado := FizzBuzz(tt.n)
		if !reflect.DeepEqual(resultado, tt.esperado) {
			t.Errorf("FizzBuzz(%d) = %v, esperado %v", tt.n, resultado, tt.esperado)
		}
	}
}

func TestEsPrimo(t *testing.T) {
	tests := []struct {
		n        int
		esperado bool
	}{
		{2, true},
		{3, true},
		{4, false},
		{5, true},
		{1, false},
		{0, false},
		{-1, false},
		{11, true},
		{25, false},
		{97, true},
	}

	for _, tt := range tests {
		resultado := EsPrimo(tt.n)
		if resultado != tt.esperado {
			t.Errorf("EsPrimo(%d) = %v, esperado %v", tt.n, resultado, tt.esperado)
		}
	}
}
