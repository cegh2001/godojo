//go:build ignore

package main

import (
	"fmt"
	"os"
)

// LeerArchivoSeguro intenta leer un archivo y devuelve su contenido.
// Si el archivo no se puede leer, envuelve el error con información adicional.
// Pista: usa os.ReadFile para leer y fmt.Errorf("%w", err) para envolver errores.
// El verbo %w crea un error envuelto que errors.Is e errors.Unwrap pueden inspeccionar.
func LeerArchivoSeguro(nombre string) (string, error) {
	// TODO: Leer el archivo con os.ReadFile(nombre)
	// TODO: Si hay error, envolverlo con fmt.Errorf("no se pudo leer %s: %w", nombre, err)
	// TODO: Si no hay error, devolver el contenido como string y nil
	return "", fmt.Errorf("no implementado")
}

// EsArchivoNoEncontrado verifica si un error es o envuelve un os.ErrNotExist.
// Usa errors.Is que recorre la cadena de errores envueltos con %w.
// Pista: errors.Is(err, os.ErrNotExist) devuelve true si err ES o ENVUELVE os.ErrNotExist.
func EsArchivoNoEncontrado(err error) bool {
	// TODO: Usar errors.Is para verificar si err envuelve os.ErrNotExist
	return false
}

// ProcesarArchivo lee un archivo y devuelve su contenido.
// Si falla, envuelve el error con contexto adicional sobre qué operación falló.
// Pista: llama a LeerArchivoSeguro y envuelve el error si falla.
func ProcesarArchivo(nombre string) (string, error) {
	// TODO: Llamar a LeerArchivoSeguro(nombre)
	// TODO: Si hay error, envolverlo con fmt.Errorf("error al procesar archivo: %w", err)
	// TODO: Si no hay error, devolver el contenido
	return "", fmt.Errorf("no implementado")
}
