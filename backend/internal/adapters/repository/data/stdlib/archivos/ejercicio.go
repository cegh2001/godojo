//go:build ignore

package main

import "os"

// LeerArchivo lee el contenido completo de un archivo como string.
// Pista: usá os.ReadFile(ruta) que devuelve ([]byte, error).
// Convertí el []byte a string con string(datos).
func LeerArchivo(ruta string) (string, error) {
	// TODO: Usar os.ReadFile(ruta) para leer el archivo
	// TODO: Si hay error, devolverlo
	// TODO: Convertir los bytes a string y devolverlos
	return "", nil
}

// EscribirArchivo escribe contenido en un archivo con permisos 0644.
// Pista: usá os.WriteFile(ruta, []byte(contenido), 0644).
func EscribirArchivo(ruta, contenido string) error {
	// TODO: Convertir contenido a []byte
	// TODO: Usar os.WriteFile para escribir
	// TODO: Devolver el error (o nil si fue exitoso)
	return nil
}

// ExisteArchivo verifica si un archivo existe en la ruta especificada.
// Pista: usá os.Stat(ruta). Si no hay error, el archivo existe.
// Si os.IsNotExist(err) es true, el archivo no existe.
func ExisteArchivo(ruta string) bool {
	// TODO: Usar os.Stat(ruta)
	// TODO: Si el error es nil, devolver true
	// TODO: Si os.IsNotExist(err), devolver false
	// TODO: Para otros errores, devolver false también
	return false
}
