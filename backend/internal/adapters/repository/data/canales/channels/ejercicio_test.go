//go:build ignore

package main

import (
	"testing"
	"time"
)

func TestEnviarMensaje_RecibirYBailar(t *testing.T) {
	tests := []struct {
		mensaje  string
		esperado string
	}{
		{"HOLA", "Mensaje recibido: HOLA"},
		{"Go es genial", "Mensaje recibido: Go es genial"},
		{"", "Mensaje recibido: "},
	}

	for _, tt := range tests {
		ch := make(chan string)
		go EnviarMensaje(tt.mensaje, ch)
		resultado := RecibirYBailar(ch)
		if resultado != tt.esperado {
			t.Errorf("Enviar/Recibir %q = %q, se esperaba %q", tt.mensaje, resultado, tt.esperado)
		}
	}
}

func TestComunicarGoroutines(t *testing.T) {
	resultado := ComunicarGoroutines()
	if resultado == "" {
		t.Error("ComunicarGoroutines() devolvió string vacío")
	}
	if resultado != "¡Hola desde gorutina!" {
		t.Errorf("ComunicarGoroutines() = %q, se esperaba %q", resultado, "¡Hola desde gorutina!")
	}
}

func TestChannel_NoBloqueaParaSiempre(t *testing.T) {
	// Verificar que el channel no se bloquea indefinidamente
	done := make(chan bool, 1)
	go func() {
		_ = ComunicarGoroutines()
		done <- true
	}()

	select {
	case <-done:
		// éxito
	case <-time.After(2 * time.Second):
		t.Fatal("ComunicarGoroutines() se bloqueó por más de 2 segundos — ¿olvidaste recibir del channel?")
	}
}

func TestRecibirYBailar_Blocking(t *testing.T) {
	// Verificar que RecibirYBailar se bloquea hasta recibir
	ch := make(chan string)
	done := make(chan bool, 1)

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch <- "mensaje demorado"
	}()

	go func() {
		_ = RecibirYBailar(ch)
		done <- true
	}()

	select {
	case <-done:
		// éxito
	case <-time.After(2 * time.Second):
		t.Fatal("RecibirYBailar se bloqueó — ¿recibe correctamente del channel?")
	}
}
