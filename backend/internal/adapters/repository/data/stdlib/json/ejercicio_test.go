//go:build ignore

package main

import (
	"strings"
	"testing"
)

func TestTareaAJSON(t *testing.T) {
	tests := []struct {
		tarea  Tarea
		campos []string // substrings que deben aparecer en el JSON
	}{
		{
			Tarea{ID: 1, Titulo: "Estudiar Go", Completada: false},
			[]string{`"id":1`, `"titulo":"Estudiar Go"`, `"completada":false`},
		},
		{
			Tarea{ID: 2, Titulo: "Hacer tests", Completada: true},
			[]string{`"id":2`, `"titulo":"Hacer tests"`, `"completada":true`},
		},
		{
			Tarea{ID: 0, Titulo: "", Completada: false},
			[]string{`"id":0`, `"titulo":""`, `"completada":false`},
		},
	}

	for _, tt := range tests {
		resultado, err := TareaAJSON(tt.tarea)
		if err != nil {
			t.Errorf("TareaAJSON(%+v) error: %v", tt.tarea, err)
			continue
		}
		for _, campo := range tt.campos {
			if !strings.Contains(resultado, campo) {
				t.Errorf("TareaAJSON(%+v) = %s — debería contener %q", tt.tarea, resultado, campo)
			}
		}
	}
}

func TestJSONATarea(t *testing.T) {
	tests := []struct {
		nombre   string
		json     string
		esperado Tarea
	}{
		{
			"tarea completa",
			`{"id":1,"titulo":"Aprender Go","completada":true}`,
			Tarea{ID: 1, Titulo: "Aprender Go", Completada: true},
		},
		{
			"tarea incompleta",
			`{"id":2,"titulo":"Practicar","completada":false}`,
			Tarea{ID: 2, Titulo: "Practicar", Completada: false},
		},
	}

	for _, tt := range tests {
		resultado, err := JSONATarea(tt.json)
		if err != nil {
			t.Errorf("JSONATarea(%s) error: %v", tt.nombre, err)
			continue
		}
		if resultado != tt.esperado {
			t.Errorf("JSONATarea(%s) = %+v, se esperaba %+v", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestJSONATarea_JSONInvalido(t *testing.T) {
	_, err := JSONATarea(`esto no es json`)
	if err == nil {
		t.Error("JSONATarea con JSON inválido debería devolver error")
	}
}

func TestListaTareasAJSON(t *testing.T) {
	tareas := []Tarea{
		{ID: 1, Titulo: "Task 1", Completada: false},
		{ID: 2, Titulo: "Task 2", Completada: true},
	}

	resultado, err := ListaTareasAJSON(tareas)
	if err != nil {
		t.Fatalf("ListaTareasAJSON error: %v", err)
	}

	// Debe contener ambos items
	camposRequeridos := []string{`"id":1`, `"id":2`, `"titulo":"Task 1"`, `"titulo":"Task 2"`}
	for _, campo := range camposRequeridos {
		if !strings.Contains(resultado, campo) {
			t.Errorf("ListaTareasAJSON debería contener %q, pero el resultado fue %s", campo, resultado)
		}
	}
}

func TestListaTareasAJSON_Vacia(t *testing.T) {
	resultado, err := ListaTareasAJSON([]Tarea{})
	if err != nil {
		t.Fatalf("ListaTareasAJSON con slice vacío error: %v", err)
	}
	if resultado != "[]" && resultado != "null" {
		t.Errorf("ListaTareasAJSON([]) = %s, se esperaba [] o null", resultado)
	}
}
