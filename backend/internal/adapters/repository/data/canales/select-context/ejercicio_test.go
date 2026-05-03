//go:build ignore

package main

import (
	"context"
	"testing"
	"time"
)

func TestEsperarConTimeout_Completado(t *testing.T) {
	resultado, err := EsperarConTimeout(100 * time.Millisecond)
	if err != nil {
		t.Errorf("EsperarConTimeout(100ms) error inesperado: %v", err)
	}
	if resultado != "completado" {
		t.Errorf("EsperarConTimeout(100ms) = %q, se esperaba \"completado\"", resultado)
	}
}

func TestEsperarConTimeout_Timeout(t *testing.T) {
	_, err := EsperarConTimeout(3 * time.Second)
	if err == nil {
		t.Error("EsperarConTimeout(3s) debería devolver error por timeout")
	}
}

func TestTrabajarConContexto_Activo(t *testing.T) {
	ctx := context.Background()
	resultado, err := TrabajarConContexto(ctx)
	if err != nil {
		t.Errorf("TrabajarConContexto(background) error inesperado: %v", err)
	}
	if resultado != "trabajo completado" {
		t.Errorf("TrabajarConContexto(background) = %q, se esperaba \"trabajo completado\"", resultado)
	}
}

func TestTrabajarConContexto_Cancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelar inmediatamente

	_, err := TrabajarConContexto(ctx)
	if err == nil {
		t.Error("TrabajarConContexto(ctx cancelado) debería devolver error")
	}
}

func TestSimularTareaLarga_Completada(t *testing.T) {
	ctx := context.Background()
	resultado, err := SimularTareaLarga(ctx, 50*time.Millisecond)
	if err != nil {
		t.Errorf("SimularTareaLarga(50ms) error inesperado: %v", err)
	}
	if resultado != "tarea completada" {
		t.Errorf("SimularTareaLarga(50ms) = %q, se esperaba \"tarea completada\"", resultado)
	}
}

func TestSimularTareaLarga_Cancelada(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := SimularTareaLarga(ctx, 2*time.Second)
	if err == nil {
		t.Error("SimularTareaLarga con ctx timeout corto debería devolver error")
	}
}

func TestEsperarConTimeout_ValorCero(t *testing.T) {
	resultado, err := EsperarConTimeout(0)
	if err != nil {
		t.Errorf("EsperarConTimeout(0) error inesperado: %v", err)
	}
	if resultado != "completado" {
		t.Errorf("EsperarConTimeout(0) = %q, se esperaba \"completado\"", resultado)
	}
}
