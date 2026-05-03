//go:build ignore

package main

import "os"

// LeerArchivo lee el contenido completo de un archivo como string.
// Pista: usa os.ReadFile(ruta) que devuelve ([]byte, error).

// Pista: usa os.WriteFile(ruta, []byte(contenido), 0644).

// Pista: usa os.Stat(ruta). Si no hay error, el archivo existe.
// Si os.IsNotExist(err) es true, el archivo no existe.
func ExisteArchivo(ruta string) bool {
	// TODO: Usar os.Stat(ruta)
	// TODO: Si el error es nil, devolver true
	// TODO: Si os.IsNotExist(err), devolver false
	// TODO: Para otros errores, devolver false también
	return false
}
