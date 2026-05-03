//go:build ignore

package main

// EnviarMensaje envía un mensaje a través de un channel de string.
// La función se bloquea hasta que alguien reciba del otro lado.
// Pista: usá el operador <- para enviar al channel: ch <- mensaje
func EnviarMensaje(mensaje string, ch chan string) {
	// TODO: Enviar mensaje al channel con ch <- mensaje
}

// RecibirYBailar recibe un mensaje de un channel de string y lo devuelve
// formateado con baile (ej. "Mensaje recibido: HOLA").
// La función se bloquea hasta que alguien envíe al channel.
// Pista: usá el operador <- para recibir del channel: msg := <-ch
func RecibirYBailar(ch chan string) string {
	// TODO: Recibir mensaje del channel con := <-ch
	// TODO: Devolver el mensaje formateado: "Mensaje recibido: " + msg
	return ""
}

// ComunicarGoroutines demuestra la comunicación entre gorutinas usando channels.
// 1. Crea un channel de string (unbuffered: make(chan string))
// 2. Lanza una gorutina que envía un saludo al channel
// 3. Recibe el mensaje del channel y lo retorna
// Pista: recordá cerrar el channel con close(ch) después de enviar
// Pista: la gorutina se lanza con la palabra clave go
func ComunicarGoroutines() string {
	// TODO: Crear un channel unbuffered: ch := make(chan string)
	// TODO: Lanzar gorutina: go func() { ch <- "¡Hola desde gorutina!" }()
	// TODO: Recibir del channel: msg := <-ch
	// TODO: Devolver msg
	return ""
}
