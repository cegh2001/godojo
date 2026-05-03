//go:build ignore

package main

// Describible es una interfaz que define un comportamiento:
// todo tipo que tenga un método Describir() string la implementa automáticamente.
type Describible interface {
	Describir() string
}

// Libro representa un libro con título y autor.
type Libro struct {
	Titulo string
	Autor  string
}

// Describir devuelve una descripción del libro.
// Al implementar este método, Libro satisface automáticamente la interfaz Describible.
func (l Libro) Describir() string {
	// TODO: Devolver "Libro: " + l.Titulo + " por " + l.Autor
	// Ejemplo: "Libro: Cien Años de Soledad por Gabriel García Márquez"
	return ""
}

// Pelicula representa una película con título y director.
type Pelicula struct {
	Titulo   string
	Director string
}

// Describir devuelve una descripción de la película.
// Al implementar este método, Pelicula también satisface Describible.
func (p Pelicula) Describir() string {
	// TODO: Devolver "Película: " + p.Titulo + " dirigida por " + p.Director
	// Ejemplo: "Película: El Padrino dirigida por Francis Ford Coppola"
	return ""
}

// ImprimirDescripcion acepta cualquier Describible y devuelve su descripción.
// Esto es polimorfismo: la función funciona con Libro, Pelicula, o cualquier
// tipo que implemente Describir() string.
func ImprimirDescripcion(d Describible) string {
	// TODO: Llamar a d.Describir() y devolver el resultado
	return ""
}
