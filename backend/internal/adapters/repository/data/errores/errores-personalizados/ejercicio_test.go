//go:build ignore

package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorDivisionCero_Error(t *testing.T) {
	e := ErrorDivisionCero{}
	mensaje := e.Error()
	if mensaje == "" {
		t.Error("ErrorDivisionCero{}.Error() no debería devolver un string vacío")
	}
	if mensaje != "no se puede dividir por cero" {
		t.Errorf("ErrorDivisionCero{}.Error() = %q, se esperaba %q", mensaje, "no se puede dividir por cero")
	}
}

func TestErrorDivisionCero_ImplementaError(t *testing.T) {
	// Verificación en tiempo de compilación
	var _ error = ErrorDivisionCero{}
}

func TestDividirSeguro(t *testing.T) {
	tests := []struct {
		a, b       float64
		esperado   float64
		debeFallar bool
	}{
		{10, 2, 5, false},
		{7, 2, 3.5, false},
		{0, 5, 0, false},
		{-10, 2, -5, false},
		{5, 0, 0, true},
		{0, 0, 0, true},
	}

	for _, tt := range tests {
		resultado, err := DividirSeguro(tt.a, tt.b)
		if tt.debeFallar {
			if err == nil {
				t.Errorf("DividirSeguro(%g, %g) debería devolver error", tt.a, tt.b)
				continue
			}
			if !EsErrorDivisionCero(err) {
				t.Errorf("DividirSeguro(%g, %g) devolvió un error que no es ErrorDivisionCero: %v", tt.a, tt.b, err)
			}
		} else {
			if err != nil {
				t.Errorf("DividirSeguro(%g, %g) error inesperado: %v", tt.a, tt.b, err)
			}
			if resultado != tt.esperado {
				t.Errorf("DividirSeguro(%g, %g) = %g, se esperaba %g", tt.a, tt.b, resultado, tt.esperado)
			}
		}
	}
}

func TestEsErrorDivisionCero(t *testing.T) {
	tests := []struct {
		err      error
		esperado bool
	}{
		{ErrorDivisionCero{}, true},
		{fmt.Errorf("otro error"), false},
		{errors.New("error genérico"), false},
		{nil, false},
	}

	for _, tt := range tests {
		resultado := EsErrorDivisionCero(tt.err)
		if resultado != tt.esperado {
			t.Errorf("EsErrorDivisionCero(%v) = %v, se esperaba %v", tt.err, resultado, tt.esperado)
		}
	}
}

func TestEsErrorDivisionCero_ConWrapping(t *testing.T) {
	// errors.As debe funcionar incluso con errores envueltos
	err := fmt.Errorf("falló la operación: %w", ErrorDivisionCero{})
	if !EsErrorDivisionCero(err) {
		t.Errorf("EsErrorDivisionCero con error envuelto debería ser true, pero dio false")
	}
}
