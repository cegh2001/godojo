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
// Pista: usá json.Marshal(t) que devuelve ([]byte, error).
// Convertí el []byte a string con string(datos).
func TareaAJSON(t Tarea) (string, error) {
	// TODO: Usar json.Marshal(t)
	// TODO: Convertir []byte a string y devolver
	return "", nil
}

// JSONATarea convierte un string JSON a una Tarea.
// Pista: usá json.Unmarshal([]byte(data), &tarea).
func JSONATarea(data string) (Tarea, error) {
	// TODO: Declarar una variable tarea de tipo Tarea
	// TODO: Usar json.Unmarshal con &tarea
	// TODO: Devolver la tarea
	return Tarea{}, nil
}

// ListaTareasAJSON convierte un slice de Tareas a JSON (string).
// Pista: usá json.Marshal igual que con una sola tarea.
func ListaTareasAJSON(tareas []Tarea) (string, error) {
	// TODO: Usar json.Marshal(tareas)
	// TODO: Convertir []byte a string y devolver
	return "", nil
}
