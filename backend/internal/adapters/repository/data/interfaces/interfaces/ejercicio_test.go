//go:build ignore

package main

import "testing"

func TestLibro_Describir(t *testing.T) {
	tests := []struct {
		titulo   string
		autor    string
		esperado string
	}{
		{
			"Cien Años de Soledad",
			"Gabriel García Márquez",
			"Libro: Cien Años de Soledad por Gabriel García Márquez",
		},
		{
			"Don Quijote",
			"Miguel de Cervantes",
			"Libro: Don Quijote por Miguel de Cervantes",
		},
		{"", "", "Libro:  por "},
	}

	for _, tt := range tests {
		l := Libro{Titulo: tt.titulo, Autor: tt.autor}
		resultado := l.Describir()
		if resultado != tt.esperado {
			t.Errorf("Libro{Titulo: %q, Autor: %q}.Describir() = %q, se esperaba %q",
				tt.titulo, tt.autor, resultado, tt.esperado)
		}
	}
}

func TestPelicula_Describir(t *testing.T) {
	tests := []struct {
		titulo   string
		director string
		esperado string
	}{
		{
			"El Padrino",
			"Francis Ford Coppola",
			"Película: El Padrino dirigida por Francis Ford Coppola",
		},
		{
			"Volver al Futuro",
			"Robert Zemeckis",
			"Película: Volver al Futuro dirigida por Robert Zemeckis",
		},
	}

	for _, tt := range tests {
		p := Pelicula{Titulo: tt.titulo, Director: tt.director}
		resultado := p.Describir()
		if resultado != tt.esperado {
			t.Errorf("Pelicula{Titulo: %q, Director: %q}.Describir() = %q, se esperaba %q",
				tt.titulo, tt.director, resultado, tt.esperado)
		}
	}
}

func TestImprimirDescripcion(t *testing.T) {
	tests := []struct {
		nombre       string
		d            Describible
		esperado     string
	}{
		{
			"Libro",
			Libro{Titulo: "1984", Autor: "George Orwell"},
			"Libro: 1984 por George Orwell",
		},
		{
			"Pelicula",
			Pelicula{Titulo: "Inception", Director: "Christopher Nolan"},
			"Película: Inception dirigida por Christopher Nolan",
		},
	}

	for _, tt := range tests {
		resultado := ImprimirDescripcion(tt.d)
		if resultado != tt.esperado {
			t.Errorf("ImprimirDescripcion(%s) = %q, se esperaba %q", tt.nombre, resultado, tt.esperado)
		}
	}
}

func TestLibro_ImplementaDescribible(t *testing.T) {
	// Verificación en tiempo de compilación: Libro debe implementar Describible
	var _ Describible = Libro{Titulo: "Test", Autor: "Test"}
}

func TestPelicula_ImplementaDescribible(t *testing.T) {
	// Verificación en tiempo de compilación: Pelicula debe implementar Describible
	var _ Describible = Pelicula{Titulo: "Test", Director: "Test"}
}
