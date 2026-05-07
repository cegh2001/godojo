# GoDojo 🥋

Plataforma interactiva de terminal (TUI) para aprender Go desde cero, combinando
TDD estricto con mentoría de IA (sensei Gemma 4 vía Gemini API) en un entorno 100% offline-first.

## Filosofía

**Dojo** = lugar de práctica disciplinada. GoDojo sigue el método socrático:
nunca te damos la solución — te guiamos con preguntas para que la descubras vos mismo.

El flujo de aprendizaje replica un coding dojo real:
1. **Elegí un tema** del roadmap de aprendizaje
2. **Leé el ejercicio** con el esqueleto de código
3. **Implementá tu solución** — los tests ya están escritos (TDD inverso)
4. **Ejecutá los tests** desde la TUI (`Ctrl+T`)
5. **Si fallan**, pedí una pista al sensei Gemma 4 (`Ctrl+H`)
6. **Si pasan**, ¡avanzá al siguiente tema!

## Roadmap de Aprendizaje

### Fase 1 — Fundamentos
| Tema | Descripción | Ejercicio |
|------|-------------|-----------|
| Variables | Declaración, tipos y uso de variables | `Saludar()` |
| Tipos de Datos | Numéricos, strings, booleanos, conversiones | `SumarEnteros()`, `Concatenar()`, `EsMayorDeEdad()` |
| Funciones | Parámetros, retornos múltiples, manejo de errores | `Dividir()`, `Estadisticas()` |
| Paquetes | Organización de código, imports, visibilidad | Importar y usar paquete `calculadora` |
| Control de Flujo | `if`, `for`, `switch` | `FizzBuzz()`, `EsPrimo()` |

### Fase 2 — Estructuras de Datos
| Tema | Descripción | Ejercicio |
|------|-------------|-----------|
| Arrays | Arrays de tamaño fijo, índices, `range` | `EncontrarMaximo()`, `InvertirArray()` |
| Slices | Slices dinámicos, `append`, `copy` | `FiltrarPares()`, `EliminarDuplicados()` |
| Maps | Mapas clave-valor, operaciones | `ContarPalabras()`, `FrecuenciaLetras()` |
| Structs | Structs, métodos, interfaces | `Rectangulo`, `Circulo`, interfaz `Figura` |

## Arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                    TUI (Bubbletea)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │ Roadmap  │  │  Topic   │  │ Exercise │  │  Hints  │ │
│  │  View    │→ │  Detail  │→ │   View   │→ │  View   │ │
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘ │
│       ↑                          ↓                      │
│       └──────── Esc ─────────────┘                      │
├─────────────────────────────────────────────────────────┤
│                 Core Services                            │
│  ┌────────────────┐  ┌────────────────┐                 │
│  │ RoadmapService │  │ExerciseService │                 │
│  └────────────────┘  └────────────────┘                 │
│  ┌────────────────┐  ┌────────────────┐                 │
│  │ ProgressService│  │  HintService   │                 │
│  └────────────────┘  └────────────────┘                 │
├─────────────────────────────────────────────────────────┤
│              Ports (Interfaces)                          │
│  ExerciseRepository │ ProgressStore │ TestRunner │        │
│                    HintProvider                          │
├─────────────────────────────────────────────────────────┤
│              Adapters (Implementations)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │ EmbedExercise│  │ JSON Progress│  │ Go Test Runner│  │
│  │    Repo      │  │    Store     │  │ (os/exec)     │  │
│  └──────────────┘  └──────────────┘  └───────────────┘  │
│  ┌──────────────────────────────┐                       │
│  │  Gemini Hint Provider (genai)│                       │
│  └──────────────────────────────┘                       │
├─────────────────────────────────────────────────────────┤
│              Domain (Entities)                           │
│  Roadmap │ Phase │ Topic │ Exercise │ Progress │ Hint    │
└─────────────────────────────────────────────────────────┘
```

### Decisiones de Arquitectura

| Decisión | Elección | Por qué |
|----------|----------|---------|
| Arquitectura | Hexagonal (Ports & Adapters) | Dominio puro, sin dependencias externas. Testeable al 100%. |
| Persistencia | JSON en `~/.godojo/progress.json` | MVP simple, human-readable, fácil de migrar. |
| TUI | Bubbletea + Lipgloss | El estándar de facto para TUIs en Go. Modelo Elm-like. |
| Tests | `go test -json` via `os/exec` | Output estructurado, deterministic, machine-parseable. |
| Embedding | `embed.FS` | Binario único, sin dependencias de sistema de archivos. |
| Gemini | Async goroutine + channels | TUI nunca se bloquea. Degradación graceful sin API key. |
| Español | Rioplatense (voseo) | Target audience: hispanohablantes. Lenguaje cálido y cercano. |

## Estructura del Proyecto

```
mi-proyecto/
├── README.md                       # Este archivo
├── .gitignore
├── backend/                        # Monorepo Go
│   ├── go.mod                      # Módulo: godojo
│   ├── cmd/
│   │   └── godojo/
│   │       ├── main.go             # Entry point — wiring de servicios
│   │       └── main_test.go        # Tests de integración del wiring
│   └── internal/
│       ├── core/
│       │   ├── domain/             # Entidades: Roadmap, Exercise, Progress…
│       │   ├── ports/              # Interfaces: ExerciseRepository, HintProvider…
│       │   └── services/           # Lógica de negocio: RoadmapService, HintService…
│       └── adapters/
│           ├── tui/                # Bubbletea: modelo, vistas, comandos, tests
│           ├── repository/         # EmbedExerciseRepo (embed.FS + generador)
│           │   └── data/           # Ejercicios embebidos (Fase 1 + Fase 2)
│           ├── runner/             # GoTestRunner (os/exec + -json parser)
│           ├── store/              # JSONProgressStore (~/.godojo/progress.json)
│           └── gemini/             # GeminiHintProvider (genai + prompt socrático)
└── ~/.godojo/                      # Creado en runtime — workspace del usuario
    ├── progress.json               # Progreso del usuario (JSON)
    └── exercises/                  # Archivos generados por ejercicio
```

## Requisitos

- **Go 1.21+** (requerido para `embed` y `go test -json`)
- **API Key de Gemini** (opcional — la app funciona sin ella, solo que sin pistas del sensei)
  - Configurá `GEMINI_API_KEY` como variable de entorno o en `.env`
  - Modelos Gemma 4 configurables por env:
    - `GODOJO_SENSEI_FAST_MODEL=gemma-4-26b-a4b-it` para hints rápidos
    - `GODOJO_SENSEI_HEAVY_MODEL=gemma-4-31b-it` para chat y tareas más pesadas
    - `GODOJO_SENSEI_MODEL=...` si querés sobreescribir solo la ruta de chat

## Cómo Ejecutar

```bash
# Clonar el repo
git clone <repo-url>
cd mi-proyecto/backend

# Ejecutar todos los tests
go test ./... -count=1

# Análisis estático
go vet ./...

# Ejecutar la TUI
go run ./cmd/godojo

# Forzar Gemma 4 por env
GODOJO_SENSEI_FAST_MODEL=gemma-4-26b-a4b-it GODOJO_SENSEI_HEAVY_MODEL=gemma-4-31b-it go run ./cmd/godojo

# Build para producción
go build -o godojo ./cmd/godojo
```

### Atajos de Teclado

| Tecla | Acción |
|-------|--------|
| `j` / `k` o `↑` / `↓` | Navegar entre items |
| `Enter` | Seleccionar / Entrar a detalle |
| `Esc` | Volver atrás |
| `Ctrl+T` | Ejecutar tests |
| `Ctrl+H` | Pedir pista al sensei (solo si fallaron los tests) |
| `q` o `Ctrl+C` | Salir |

## Testing

El proyecto tiene **cobertura completa de tests** con el protocolo TDD estricto:

| Capa | Tests | Enfoque |
|------|-------|---------|
| Dominio | 42 | Constructores, validación, transiciones de estado |
| Servicios | 44 | Mocks de puertos, table-driven |
| Adaptadores | 46 | Golden files, temp dirs, integration |
| TUI | 56 | Modelo, vistas, comandos |
| Main | 4 | Wiring, degradación sin API key |
| Integración | 10 | Flujo completo, navegación, progreso |
| **Total** | **202+** | Table-driven, go test -count=1 |

```bash
cd backend
go test ./... -count=1   # Ejecutar todos los tests
go test ./... -cover     # Con cobertura
go vet ./...             # Análisis estático

# Smoke tests live contra la API real (requiere GEMINI_API_KEY)
GODOJO_GEMINI_LIVE_TEST=1 go test ./internal/adapters/gemini -run Live -count=1
```

## Cómo Contribuir

1. Los ejercicios se definen en `backend/internal/adapters/repository/data/`
2. Cada ejercicio tiene: `ejercicio.go` (esqueleto), `ejercicio_test.go` (tests), `README.md` (instrucciones en español)
3. El manifiesto de ejercicios está en `embed_exercise_repo.go`
4. Para agregar una nueva fase: crear directorio, agregar ejercicios, actualizar el manifiesto y `RoadmapService`

## Licencia

MIT — Aprendé, compartí, contribuí.
