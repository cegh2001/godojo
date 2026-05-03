//go:build ignore

package main

import (
	"os"
	"testing"
)

func TestLeerArchivo(t *testing.T) {
	// Crear archivo temporal
	archivo := "test_leer_temp.txt"
	contenido := "Golang es divertido"
	if err := os.WriteFile(archivo, []byte(contenido), 0644); err != nil {
		t.Fatalf("No se pudo crear archivo de prueba: %v", err)
	}
	defer os.Remove(archivo)

	resultado, err := LeerArchivo(archivo)
	if err != nil {
		t.Fatalf("LeerArchivo(%q) error inesperado: %v", archivo, err)
	}
	if resultado != contenido {
		t.Errorf("LeerArchivo(%q) = %q, se esperaba %q", archivo, resultado, contenido)
	}
}

func TestLeerArchivo_NoExiste(t *testing.T) {
	_, err := LeerArchivo("archivo_inexistente_abc.txt")
	if err == nil {
		t.Error("LeerArchivo debería devolver error para archivo inexistente")
	}
}

func TestEscribirArchivo(t *testing.T) {
	archivo := "test_escribir_temp.txt"
	contenido := "Aprendiendo Go"
	defer os.Remove(archivo)

	err := EscribirArchivo(archivo, contenido)
	if err != nil {
		t.Fatalf("EscribirArchivo(%q) error: %v", archivo, err)
	}

	// Verificar que el contenido se escribió correctamente
	leido, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatalf("No se pudo leer archivo escrito: %v", err)
	}
	if string(leido) != contenido {
		t.Errorf("EscribirArchivo: el contenido leído (%q) no coincide con el esperado (%q)", string(leido), contenido)
	}
}

func TestExisteArchivo(t *testing.T) {
	tests := []struct {
		ruta     string
		crear    bool
		esperado bool
	}{
		{"test_existe_temp.txt", true, true},
		{"no_existe_xyz.txt", false, false},
	}

	for _, tt := range tests {
		if tt.crear {
			if err := os.WriteFile(tt.ruta, []byte("test"), 0644); err != nil {
				t.Fatalf("No se pudo crear archivo: %v", err)
			}
			defer os.Remove(tt.ruta)
		}

		resultado := ExisteArchivo(tt.ruta)
		if resultado != tt.esperado {
			t.Errorf("ExisteArchivo(%q) = %v, se esperaba %v", tt.ruta, resultado, tt.esperado)
		}
	}
}
