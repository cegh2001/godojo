//go:build ignore

package main

import (
	"fmt"
	"sync"
)

// ContarHasta retorna un string con la secuencia de conteo de 1 hasta n.
// Ejemplo: ContarHasta(3, "A") → "A: 1, A: 2, A: 3"
// El parámetro nombre identifica al "contador" en el mensaje.
// Pista: usa un bucle for y fmt.Sprintf para construir cada línea.
func ContarHasta(n int, nombre string) string {
	// TODO: Usar un bucle for i := 1; i <= n; i++
	// TODO: Acumular los mensajes en una variable usando fmt.Sprintf
	// TODO: ¡No uses gorutinas en esta función! Es una función auxiliar normal
	return ""
}

// EjecutarConcurrente lanza 2 gorutinas que cuentan hasta n concurrentemente.
// Ambas gorutinas llaman a ContarHasta y almacenan el resultado.
// Usa sync.WaitGroup para esperar que ambas terminen y sync.Mutex para
// proteger el acceso al slice compartido de resultados.
//
// Pista de estructura:
//   1. Crear un WaitGroup y un Mutex
//   2. Crear un slice para resultados
//   3. wg.Add(2)
//   4. Lanzar gorutina 1: defer wg.Done(), mutex.Lock(), append resultado, mutex.Unlock()
//   5. Lanzar gorutina 2: igual
//   6. wg.Wait() para esperar que ambas terminen
//   7. Devolver el slice de resultados
func EjecutarConcurrente(n int) []string {
	// TODO: Crear sync.WaitGroup y sync.Mutex
	// TODO: Crear slice de resultados (var resultados []string)
	// TODO: wg.Add(2)
	// TODO: Lanzar gorutina 1 que cuente con nombre "Gorutina 1"
	// TODO: Lanzar gorutina 2 que cuente con nombre "Gorutina 2"
	// TODO: wg.Wait()
	// TODO: Devolver resultados
	return nil
}
