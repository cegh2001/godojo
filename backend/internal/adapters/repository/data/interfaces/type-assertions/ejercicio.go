//go:build ignore

package main

// Describible es una interfaz que define el comportamiento Describir.
type Describible interface {
	Describir() string
}

// Libro representa un libro con título y autor.
type Libro struct {
	Titulo string
	Autor  string
}

// Describir devuelve una descripción del libro.
func (l Libro) Describir() string {
	return "Libro: " + l.Titulo + " por " + l.Autor
}

// Pelicula representa una película con título y director.
type Pelicula struct {
	Titulo   string
	Director string
}

// Describir devuelve una descripción de la película.
func (p Pelicula) Describir() string {
	return "Película: " + p.Titulo + " dirigida por " + p.Director
}

// EsLibro verifica si un Describible es específicamente un Libro.
// Pista: usa una type assertion con el patrón "comma-ok".

// Pista: usa un type switch con casos para Libro y Pelicula.

// Pista: usa un type switch como en ObtenerTitulo.
func TipoDeDescribible(d Describible) string {
	// TODO: Usar type switch sobre d.(type)
	return ""
}
