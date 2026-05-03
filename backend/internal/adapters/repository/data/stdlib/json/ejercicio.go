//go:build ignore

package main

import "encoding/json"

// Tarea representa una tarea con ID, título y estado de completado.
// Las etiquetas json (struct tags) le dicen a encoding/json cómo mapear
// los campos al serializar/deserializar.
type Tarea struct {
	ID         int    `json:"id"`
	Titulo     string `json:"titulo"`
	Completada bool   `json:"completada"`
}

// TareaAJSON convierte una Tarea a su representación JSON (string).
// Pista: usa json.Marshal(t) que devuelve ([]byte, error).

// Pista: usa json.Unmarshal([]byte(data), &tarea).

// Pista: usa json.Marshal igual que con una sola tarea.
func ListaTareasAJSON(tareas []Tarea) (string, error) {
	// TODO: Usar json.Marshal(tareas)
	// TODO: Convertir []byte a string y devolver
	return "", nil
}
