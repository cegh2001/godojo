//go:build ignore

package main

import (
	"errors"
	"os"
	"testing"
)

func TestLeerArchivoSeguro_Existe(t *testing.T) {
	// Crear un archivo temporal para la prueba
	archivo := "test_archivo_temp.txt"
	contenido := "Hola, Go!"
	if err := os.WriteFile(archivo, []byte(contenido), 0644); err != nil {
		t.Fatalf("No se pudo crear archivo de prueba: %v", err)
	}
	defer os.Remove(archivo)

	resultado, err := LeerArchivoSeguro(archivo)
	if err != nil {
		t.Fatalf("LeerArchivoSeguro(%q) error inesperado: %v", archivo, err)
	}
	if resultado != contenido {
		t.Errorf("LeerArchivoSeguro(%q) = %q, se esperaba %q", archivo, resultado, contenido)
	}
}

func TestLeerArchivoSeguro_NoExiste(t *testing.T) {
	_, err := LeerArchivoSeguro("archivo_que_no_existe_xyz.txt")
	if err == nil {
		t.Error("LeerArchivoSeguro debería devolver error para archivo inexistente")
	}
}

func TestEsArchivoNoEncontrado(t *testing.T) {
	tests := []struct {
		nombre   string
		err      error
		esperado bool
	}{
		{"os.ErrNotExist directo", os.ErrNotExist, true},
		{"error envuelto", ProcesarArchivo("no_existe.txt"), true}, // processFilePath may return wrapped error
		{"otro error", errors.New("error cualquiera"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		resultado := EsArchivoNoEncontrado(tt.err)
		if resultado != tt.esperado {
			t.Errorf("EsArchivoNoEncontrado(%s) = %v, se esperaba %v", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestProcesarArchivo(t *testing.T) {
	// Crear archivo temporal
	archivo := "test_procesar_temp.txt"
	contenido := "Contenido de prueba"
	if err := os.WriteFile(archivo, []byte(contenido), 0644); err != nil {
		t.Fatalf("No se pudo crear archivo de prueba: %v", err)
	}
	defer os.Remove(archivo)

	resultado, err := ProcesarArchivo(archivo)
	if err != nil {
		t.Fatalf("ProcesarArchivo(%q) error inesperado: %v", archivo, err)
	}
	if resultado != contenido {
		t.Errorf("ProcesarArchivo(%q) = %q, se esperaba %q", archivo, resultado, contenido)
	}
}

func TestProcesarArchivo_NoExiste(t *testing.T) {
	_, err := ProcesarArchivo("este_archivo_no_existe_nunca.txt")
	if err == nil {
		t.Error("ProcesarArchivo debería devolver error para archivo inexistente")
	}
	// Verificar que el error envuelve os.ErrNotExist
	if !EsArchivoNoEncontrado(err) {
		t.Logf("ProcesarArchivo devolvió error, pero no envuelve os.ErrNotExist: %v", err)
	}
}

func TestErrorWrapping_Cadena(t *testing.T) {
	// Verificar que el error envuelto mantiene la cadena de wrapping
	_, err := ProcesarArchivo("no_existe.txt")
	if err == nil {
		t.Fatal("Se esperaba un error")
	}

	// errors.Is debe encontrar os.ErrNotExist en la cadena
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("errors.Is(err, os.ErrNotExist) debería ser true para un archivo inexistente, pero dio false. Error: %v", err)
	}
}
