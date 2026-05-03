//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"
)

// EsperarConTimeout espera un tiempo determinado y devuelve "completado".
// Si el tiempo es mayor a 2 segundos, devuelve un error por timeout.
// Pista: usá select con time.After para implementar el timeout.
//
// Estructura del select:
//   select {
//   case <-time.After(tiempo):
//       return "completado", nil
//   case <-time.After(2 * time.Second):
//       return "", fmt.Errorf("timeout")
//   }
func EsperarConTimeout(tiempo time.Duration) (string, error) {
	// TODO: Usar select con dos casos de time.After
	// TODO: Caso 1: esperar 'tiempo' y devolver "completado"
	// TODO: Caso 2: timeout a los 2 segundos con error
	return "", fmt.Errorf("no implementado")
}

// TrabajarConContexto verifica si un contexto fue cancelado.
// Si el contexto está activo, devuelve "trabajo completado".
// Si el contexto fue cancelado (ctx.Done()), devuelve el error del contexto.
// Pista: usá select con ctx.Done() para verificar cancelación.
//
// Estructura:
//   select {
//   case <-ctx.Done():
//       return "", ctx.Err()
//   default:
//       return "trabajo completado", nil
//   }
func TrabajarConContexto(ctx context.Context) (string, error) {
	// TODO: Usar select con ctx.Done()
	// TODO: Si el contexto se canceló, devolver error
	// TODO: Si no, devolver "trabajo completado"
	return "", fmt.Errorf("no implementado")
}

// SimularTareaLarga simula una tarea que toma una duración específica.
// Debe respetar la cancelación del contexto: si el contexto se cancela
// antes de que termine la duración, devolver el error del contexto.
// Si la duración transcurre sin cancelación, devolver "tarea completada".
// Pista: combiná time.After y ctx.Done() en un select.
func SimularTareaLarga(ctx context.Context, duracion time.Duration) (string, error) {
	// TODO: Usar select con dos casos:
	// TODO: Caso 1: <-time.After(duracion) → "tarea completada"
	// TODO: Caso 2: <-ctx.Done() → error del contexto
	return "", fmt.Errorf("no implementado")
}
