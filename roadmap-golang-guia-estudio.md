# Guía de estudio de Go basada en roadmap.sh

Documento sintetizado a partir de https://roadmap.sh/golang. No copia el contenido completo; organiza los títulos principales y lo convierte en un plan de estudio práctico con recursos.

## Enlaces principales

- Roadmap principal: https://roadmap.sh/golang
- Proyectos del roadmap: https://roadmap.sh/golang/projects
- AI Tutor del roadmap: https://roadmap.sh/ai/roadmap-chat/golang
- Volver a todos los roadmaps: https://roadmap.sh/roadmaps
- Documentación oficial de Go: https://go.dev/doc/
- Tutorial interactivo de Go: https://go.dev/tour/
- Paquetes y APIs: https://pkg.go.dev/

## Extracto de títulos del roadmap

### 1. Introducción

- Go Developer
- What is Golang?
- Related Roadmaps
  - Backend Roadmap
  - DevOps Roadmap
  - Docker Roadmap
  - Kubernetes Roadmap
  - System Design
  - Software Design & Architecture
- Setting up the Environment
- Hello World in Go
- `go` command
- Backend

### 2. Bases del lenguaje

- Language Basics
- Variables & Constants
  - var vs :=
  - Zero Values
  - const and iota
  - Scope and Shadowing
- Data Types
  - Boolean
  - Numeric Types
    - Integers (Signed, Unsigned)
    - Floating Points
    - Complex Numbers
  - Runes
  - Strings
    - Raw String Literals
    - Interpreted String Literals
  - Type Conversion
- Commands & Docs

### 3. Tipos compuestos y flujo de control

- Composite Types
  - Arrays
  - Slices
  - Capacity and Growth
  - make()
  - Slice to Array Conversion
  - Array to Slice Conversion
  - Strings
  - Maps
  - Comma-Ok Idiom
  - Structs
  - Struct Tags & JSON
  - Embedding Structs
- Conditionals
  - if
  - if-else
  - switch
- Loops
  - for loop
  - for range
  - Iterating Maps
  - Iterating Strings
  - break
  - continue
  - goto (discouraged)

### 4. Funciones, punteros y memoria

- Functions
  - Functions Basics
  - Variadic Functions
  - Multiple Return Values
  - Anonymous Functions
  - Closures
  - Named Return Values
  - Call by Value
- Pointers
  - Pointers Basics
  - Pointers with Structs
  - With Maps & Slices
- Memory Management
  - Garbage Collection
  - Get a Brief Overview

### 5. Métodos, interfaces y genéricos

- Methods and Interfaces
  - Methods vs Functions
  - Pointer Receivers
  - Value Receivers
  - Interfaces
  - Empty Interfaces
  - Embedding Interfaces
  - Type Assertions
  - Type Switch
  - Interfaces Basics
- Generics
  - Why Generics?
  - Generic Functions
  - Generic Types / Interfaces
  - Type Constraints
  - Type Inference

### 6. Manejo de errores y organización del código

- Error Handling
  - Error Handling Basics
  - `error` interface
  - errors.New
  - fmt.Errorf
  - Wrapping/Unwrapping Errors
  - Sentinel Errors
  - `panic` and `recover`
  - Stack Traces & Debugging
- Code Organization
  - Modules & Dependencies
  - go mod init
  - go mod tidy
  - go mod vendor
  - Packages
  - Package Import Rules
  - Using 3rd Party Packages
  - Publishing Modules

### 7. Concurrencia

- Goroutines
- Channels
  - Buffered vs Unbuffered
  - Select Statement
  - Worker Pools
- `sync` Package
  - Mutexes
  - WaitGroups
- `context` Package
  - Deadlines & Cancellations
  - Common Usecases
- Concurrency Patterns
  - fan-in
  - fan-out
  - pipeline
- Race Detection

### 8. Librería estándar, pruebas y ecosistema

- Standard Library
  - I/O & File Handling
  - flag
  - time
  - encoding/json
  - os
  - bufio
  - slog
  - regexp
  - go:embed for embedding
- Testing & Benchmarking
  - `testing` package basics
  - Table-driven Tests
  - Mocks and Stubs
  - `httptest` for HTTP Tests
  - Benchmarks
  - Coverage
- Ecosystem & Popular Libraries
  - Building CLIs
  - Cobra
  - urfave/cli
  - bubbletea
  - Web Development
  - net/http (standard)
  - Frameworks (Optional)
    - gin
    - echo
    - fiber
    - beego
  - gRPC & Protocol Buffers
  - ORMs & DB Access
    - pgx
    - GORM
  - Logging
    - Zerolog
    - Zap
  - Realtime Communication
    - Melody
    - Centrifugo

### 9. Toolchain, despliegue y temas avanzados

- Go Toolchain and Tools
  - go run
  - Core Go Commands
    - go build
    - go install
    - go fmt
    - go mod
    - go test
    - go clean
    - go doc
    - go version
  - Code Generation / Build Tags
    - go generate
    - Build Tags
  - Code Quality and Analysis
    - go vet
    - goimports
    - Linters
      - revive
      - staticcheck
      - golangci-lint
  - Security
    - govulncheck
  - Performance and Debugging
    - pprof
    - trace
    - Race Detector
- Deployment & Tooling
  - Cross-compilation
  - Building Executables
- Advanced Topics
  - Memory Mgmt. in Depth
  - Escape Analysis
  - Reflection
  - Unsafe Package
  - Build Constraints & Tags
  - CGO Basics
  - Compiler & Linker Flags
  - Plugins & Dynamic Loading
- DevOps
  - Docker
  - Kubernetes
- Concurrency

### 10. Cierre

- Frequently Asked Questions
- Additional Links

## Guía de estudio recomendada

### Paso 1: Fundamentos mínimos

Objetivo: entender sintaxis, tipos, control de flujo y funciones sin saltar a frameworks.

Recursos:

- https://go.dev/tour/
- https://go.dev/doc/effective_go
- https://pkg.go.dev/builtin

Práctica sugerida:

- Resolver pequeños ejercicios con variables, condicionales, slices, maps y structs.
- Escribir funciones con múltiples retornos y manejo básico de errores.

### Paso 2: Organización y herramientas

Objetivo: trabajar como un proyecto real de Go.

Recursos:

- https://go.dev/doc/modules/managing-dependencies
- https://go.dev/doc/modules/layout
- https://pkg.go.dev/cmd/go

Práctica sugerida:

- Crear un módulo con `go mod init`.
- Separar paquetes por responsabilidad.
- Probar `go fmt`, `go test`, `go vet` y `go doc`.

### Paso 3: Concurrencia

Objetivo: dominar goroutines, channels, context y sincronización.

Recursos:

- https://go.dev/blog/pipelines
- https://go.dev/blog/context
- https://pkg.go.dev/sync
- https://pkg.go.dev/context

Práctica sugerida:

- Implementar worker pools y pipelines simples.
- Detectar condiciones de carrera con `go test -race`.

### Paso 4: Testing y calidad

Objetivo: escribir pruebas útiles, legibles y mantenibles.

Recursos:

- https://go.dev/doc/tutorial/add-a-test
- https://pkg.go.dev/testing
- https://pkg.go.dev/net/http/httptest

Práctica sugerida:

- Usar tests table-driven.
- Cubrir casos felices, errores y bordes.
- Medir cobertura y ajustar diseño para testear mejor.

### Paso 5: Ecosistema aplicado

Objetivo: elegir herramientas según el problema, no por moda.

Recursos:

- https://github.com/spf13/cobra
- https://github.com/urfave/cli
- https://github.com/charmbracelet/bubbletea
- https://pkg.go.dev/net/http
- https://pkg.go.dev/google.golang.org/grpc

Práctica sugerida:

- Hacer una CLI pequeña.
- Construir una API HTTP simple.
- Probar logging estructurado y acceso a datos.

### Paso 6: Avanzado y producción

Objetivo: entender optimización, seguridad y despliegue.

Recursos:

- https://go.dev/blog/escape-analysis
- https://pkg.go.dev/cmd/cgo
- https://pkg.go.dev/cmd/link
- https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck

Práctica sugerida:

- Compilar binarios para distintas plataformas.
- Revisar escape analysis y perfiles con `pprof`.
- Probar build tags, embebido de archivos y `go generate`.

## Ruta corta de 4 semanas

### Semana 1

- Sintaxis básica, tipos, control de flujo, funciones y punteros.

### Semana 2

- Structs, interfaces, errores, paquetes y módulos.

### Semana 3

- Concurrencia, context, sync, channels y race detector.

### Semana 4

- Testing, estándar de biblioteca, CLI o API pequeña y tooling.

## Proyecto de práctica

Construye una herramienta en Go que:

- Lea y escriba archivos JSON.
- Exponga una API HTTP pequeña o una CLI.
- Use goroutines para trabajo concurrente.
- Tenga tests table-driven.
- Incluya logging, contexto y manejo de errores.

## Nota final

El roadmap también incluye FAQ y enlaces extra. Si quieres, puedo convertir este documento en una versión más corta tipo checklist o en una versión más técnica con ejercicios por cada punto.