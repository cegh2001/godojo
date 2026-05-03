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
// Pista: usá una type assertion con el patrón "comma-ok".
// El patrón comma-ok evita pánico si la aserción falla:
//   valor, ok := d.(Libro)
//   if ok { ... }
func EsLibro(d Describible) bool {
	// TODO: Usar type assertion d.(Libro) con el patrón comma-ok
	// TODO: Devolver true si es un Libro, false si no
	return false
}

// ObtenerTitulo devuelve el título de cualquier Describible.
// Si es un Libro, devuelve l.Titulo.
// Si es una Pelicula, devuelve p.Titulo.
// En otro caso, devuelve "Desconocido".
// Pista: usá un type switch con casos para Libro y Pelicula.
//   switch v := d.(type) {
//   case Libro:
//       return v.Titulo
//   case Pelicula:
//       return v.Titulo
//   default:
//       return "Desconocido"
//   }
func ObtenerTitulo(d Describible) string {
	// TODO: Usar type switch sobre d.(type) con casos para Libro y Pelicula
	return ""
}

// TipoDeDescribible devuelve el tipo concreto como string:
// "Libro", "Pelicula" o "Desconocido".
// Pista: usá un type switch como en ObtenerTitulo.
func TipoDeDescribible(d Describible) string {
	// TODO: Usar type switch sobre d.(type)
	return ""
}
