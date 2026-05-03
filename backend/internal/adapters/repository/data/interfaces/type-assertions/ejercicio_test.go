//go:build ignore

package main

import "testing"

func TestEsLibro(t *testing.T) {
	tests := []struct {
		nombre   string
		d        Describible
		esperado bool
	}{
		{
			"Libro",
			Libro{Titulo: "1984", Autor: "Orwell"},
			true,
		},
		{
			"Pelicula",
			Pelicula{Titulo: "Matrix", Director: "Wachowski"},
			false,
		},
		{
			"Libro vacío",
			Libro{},
			true,
		},
	}

	for _, tt := range tests {
		resultado := EsLibro(tt.d)
		if resultado != tt.esperado {
			t.Errorf("EsLibro(%s) = %v, se esperaba %v", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestObtenerTitulo(t *testing.T) {
	tests := []struct {
		nombre   string
		d        Describible
		esperado string
	}{
		{
			"Libro",
			Libro{Titulo: "El Principito", Autor: "Saint-Exupéry"},
			"El Principito",
		},
		{
			"Pelicula",
			Pelicula{Titulo: "Interestelar", Director: "Nolan"},
			"Interestelar",
		},
		{
			"Libro vacío",
			Libro{},
			"",
		},
		{
			"Pelicula vacía",
			Pelicula{},
			"",
		},
	}

	for _, tt := range tests {
		resultado := ObtenerTitulo(tt.d)
		if resultado != tt.esperado {
			t.Errorf("ObtenerTitulo(%s) = %q, se esperaba %q", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestTipoDeDescribible(t *testing.T) {
	tests := []struct {
		nombre   string
		d        Describible
		esperado string
	}{
		{"Libro", Libro{Titulo: "Test", Autor: "Test"}, "Libro"},
		{"Pelicula", Pelicula{Titulo: "Test", Director: "Test"}, "Pelicula"},
	}

	for _, tt := range tests {
		resultado := TipoDeDescribible(tt.d)
		if resultado != tt.esperado {
			t.Errorf("TipoDeDescribible(%s) = %q, se esperaba %q", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestCommaOk_NoPanic(t *testing.T) {
	// Verificar que EsLibro no hace pánico con valores válidos e inválidos
	_ = EsLibro(Libro{Titulo: "X", Autor: "Y"})
	_ = EsLibro(Pelicula{Titulo: "Z", Director: "W"})
}
