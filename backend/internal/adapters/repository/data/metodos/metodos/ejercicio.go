//go:build ignore

package main

// Persona representa una persona con nombre y edad.
type Persona struct {
	Nombre string
	Edad   int
}

// Saludar devuelve un saludo personalizado con el nombre de la persona.
// Ejemplo: Para Persona{Nombre: "Ana"}, devuelve "¡Hola, soy Ana!"
// Este método usa value receiver porque solo LEE datos, no los modifica.
func (p Persona) Saludar() string {
	// TODO: Devolver "¡Hola, soy " + p.Nombre + "!"
	return ""
}

// CumplirAños incrementa la edad de la persona en 1.
// Este método usa pointer receiver (*Persona) porque MODIFICA el struct.
func (p *Persona) CumplirAños() {
	// TODO: Incrementar p.Edad en 1
}

// EsMayor determina si la persona es mayor de edad (18 años o más).
// Este método usa value receiver porque solo LEE la edad.
func (p Persona) EsMayor() bool {
	// TODO: Devolver true si p.Edad >= 18
	return false
}
