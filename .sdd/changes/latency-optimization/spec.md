# Delta Specs: latency-optimization

## 1. MODIFIED: `genai-provider-model-selection` — Restore fast model routing

### What Changes

The `GenaiProvider` currently ignores `GODOJO_SENSEI_FAST_MODEL` and routes ALL requests through the heavy model (`GODOJO_SENSEI_MODEL`, default `gemma-4-31b-it`). This is the primary cause of the 17.67s latency regression from `sensei-tools-upgrade`.

Restore model selection logic that the old `ChatProvider` had via `forTextRequest()`:

| Condition | Model | Env Var | Default |
|---|---|---|---|
| `tools == nil` AND flash configured AND simple intent | Flash | `GODOJO_SENSEI_FLASH_MODEL` | empty (disabled) |
| `tools == nil` AND NOT (flash + simple intent) | Fast | `GODOJO_SENSEI_FAST_MODEL` | `gemma-4-26b-a4b-it` |
| `tools != nil` (tool-enabled) | Heavy | `GODOJO_SENSEI_MODEL` | `gemma-4-31b-it` |
| `SendFunctionResponse` | Heavy | `GODOJO_SENSEI_MODEL` | `gemma-4-31b-it` |

### Files affected

- `backend/internal/adapters/genai/genai_provider.go` — add model resolution env vars + selection logic
- `.env.example` — document `GODOJO_SENSEI_FLASH_MODEL`

### Requirements

- **REQ-MS-01**: Fast model env var read and defaulted — `GODOJO_SENSEI_FAST_MODEL` defaults to `gemma-4-26b-a4b-it`
- **REQ-MS-02**: Text-only requests (`tools == nil`) use fast model
- **REQ-MS-03**: Tool-enabled requests (`tools != nil`) use heavy model (unchanged behavior)
- **REQ-MS-04**: `SendFunctionResponse` always uses heavy model (unchanged)
- **REQ-MS-05**: Flash model for detected simple intents when `GODOJO_SENSEI_FLASH_MODEL` is configured
- **REQ-MS-06**: Simple intent detection — matches greetings (hola, holi, chau, buenass, qué tal) and capabilities questions (qué podés hacer, quién sos, cómo funcionas, que podes hacer)
- **REQ-MS-07**: Graceful fallback — if fast model env var is empty or resolves to an invalid model name, use heavy model (don't crash)
- **REQ-MS-08**: Spanish-language system prompt preserved on all model tiers (fast, flash, heavy)

### Scenarios

**Scenario MS-01: Text-only greeting uses fast model**
```
GIVEN a GenaiProvider with GODOJO_SENSEI_FAST_MODEL=gemma-4-26b-a4b-it
  AND tools == nil
  AND user message is "holi"
THEN SendMessage uses model "gemma-4-26b-a4b-it"
```

**Scenario MS-02: Tool-enabled request uses heavy model**
```
GIVEN a GenaiProvider with GODOJO_SENSEI_MODEL=gemma-4-31b-it
  AND tools has 5 tool declarations
  AND user message is "creá un archivo"
THEN SendMessage uses model "gemma-4-31b-it"
```

**Scenario MS-03: Function response uses heavy model (same as MS-02)**
```
GIVEN a GenaiProvider calling SendFunctionResponse
THEN heavy model is used regardless of tools
```

**Scenario MS-04: Flash model for detected greeting when configured**
```
GIVEN a GenaiProvider with GODOJO_SENSEI_FLASH_MODEL=gemini-2.5-flash
  AND GODOJO_SENSEI_FAST_MODEL=gemma-4-26b-a4b-it
  AND tools == nil
  AND user message is "hola"
THEN SendMessage uses model "gemini-2.5-flash"
```

**Scenario MS-05: Non-greeting text uses fast model even with flash configured**
```
GIVEN a GenaiProvider with GODOJO_SENSEI_FLASH_MODEL=gemini-2.5-flash
  AND tools == nil
  AND user message is "explicame que son las goroutines"
THEN SendMessage uses model "gemma-4-26b-a4b-it" (fast, not flash)
```

**Scenario MS-06: No flash model configured — fast model used for text**
```
GIVEN a GenaiProvider without GODOJO_SENSEI_FLASH_MODEL set
  AND tools == nil
  AND user message is "holi"
THEN SendMessage uses model "gemma-4-26b-a4b-it" (fast, not flash — flash disabled)
```

**Scenario MS-07: Empty fast model falls back to heavy**
```
GIVEN a GenaiProvider with GODOJO_SENSEI_FAST_MODEL=""
  AND tools == nil
THEN SendMessage uses heavy model (GODOJO_SENSEI_MODEL or default gemma-4-31b-it)
```

**Scenario MS-08: System prompt preserved on fast model**
```
GIVEN the system prompt uses voseo Spanish (e.g., "Usá español neutro latinoamericano")
  AND tools == nil
  AND the model is resolved to fast model
THEN the config's SystemInstruction still contains the full Spanish rioplatense prompt
```

---

## 2. NEW: `sensei-provider-streaming` — Streaming via GenerateContentStream

### What Changes

Add streaming support to `SenseiProvider` interface and implement in `GenaiProvider` using the genai SDK's `GenerateContentStream`. Streaming is ONLY for text-only requests (`tools == nil`); tool-enabled requests continue using the existing non-streaming `SendMessage`.

### New types

```go
// StreamChunk represents a single chunk from a streaming response.
type StreamChunk struct {
    Text         string              // partial text from the stream
    FunctionCall *domain.FunctionCall // present if the chunk contains a function call (future use)
    Done         bool                // true for the final chunk
    Error        error               // non-nil if a stream error occurred
}
```

### Interface addition

```go
type SenseiProvider interface {
    SendMessage(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) ([]domain.ContentPart, error)
    SendFunctionResponse(ctx context.Context, history []chatstore.ChatMessage, callID string, name string, result interface{}) ([]domain.ContentPart, error)
    SendMessageStream(ctx context.Context, systemPrompt string, history []chatstore.ChatMessage, tools []domain.ToolDeclaration) (<-chan StreamChunk, error)  // NEW
}
```

### Implementation approach

`GenaiProvider.SendMessageStream` calls `p.client.Models.GenerateContentStream(ctx, model, contents, config)`, consumes the `iter.Seq2[*GenerateContentResponse, error]`, and pushes chunks into a buffered Go channel:

```
iter.Seq2 → for each (resp, err):
  if err → push StreamChunk{Error: err}, close channel, return
  if resp has candidates[0].content.parts →
    for each part (skip thought):
      push StreamChunk{Text: part.Text}
  if resp.Candidates[0].FinishReason == finishReasonStop →
    push StreamChunk{Done: true}, close channel
```

### Files affected

- `backend/internal/core/ports/sensei_provider.go` — add `StreamChunk` type + `SendMessageStream` to interface
- `backend/internal/adapters/genai/genai_provider.go` — implement `SendMessageStream`
- `backend/internal/core/domain/sensei_types.go` — optionally add `StreamChunk` here (design decision: port or domain)

### Requirements

- **REQ-SS-01**: `SendMessageStream` method on `SenseiProvider` interface (additive, non-breaking — existing callers unchanged)
- **REQ-SS-02**: `GenaiProvider` implements streaming via `GenerateContentStream` from `google.golang.org/genai`
- **REQ-SS-03**: `StreamChunk` type with `Text`, `FunctionCall`, `Done`, `Error` fields
- **REQ-SS-04**: Streaming produces at least one chunk before timeout (even for empty responses, send a Done chunk)
- **REQ-SS-05**: Streaming gracefully handles model errors — non-nil `Error` field, then close channel
- **REQ-SS-06**: Non-streaming `SendMessage` path unchanged (backward compatible)
- **REQ-SS-07**: Streaming is ONLY for text-only requests (`tools == nil`); `tools != nil` returns error or falls through to existing path

### Scenarios

**Scenario SS-01: Streaming text-only response**
```
GIVEN a GenaiProvider with a valid API key
  AND tools == nil
WHEN SendMessageStream is called with a greeting
THEN the returned channel produces one or more StreamChunk with non-empty Text
  AND the last chunk has Done == true
  AND the channel is closed after the Done chunk
```

**Scenario SS-02: Streaming error handling**
```
GIVEN a GenaiProvider with an invalid API key
WHEN SendMessageStream is called
THEN the channel yields exactly one chunk with non-nil Error
  AND Done == false
  AND the channel is closed
```

**Scenario SS-03: Streaming with tools returns error**
```
GIVEN a GenaiProvider
WHEN SendMessageStream is called with tools != nil
THEN it returns (nil, error) — streaming not supported for tool-enabled requests
```

**Scenario SS-04: Non-streaming path unchanged**
```
GIVEN a GenaiProvider
WHEN SendMessage (non-streaming) is called
THEN it behaves identically to before — no regression
```

---

## 3. MODIFIED: `sensei-agent-loop-streaming` — Use streaming in agent loop

### What Changes

Modify `runAgentLoop` in `SenseiService` to use `SendMessageStream` when available AND `tools == nil`. Stream chunks are forwarded through the status channel as they arrive, enabling progressive display. Tool-enabled requests continue using the existing non-streaming path.

### Agent loop flow with streaming

```
runAgentLoop:
  if toolDeclarations == nil AND provider supports streaming:
    chunkCh, err := provider.SendMessageStream(...)
    if err → send error on statusCh, return
    
    var fullText strings.Builder
    for chunk := range chunkCh:
      if chunk.Error != nil → send error on statusCh, return
      if chunk.Text != "":
        fullText.WriteString(chunk.Text)
        statusCh <- chunk.Text       // forward as progress update
      if chunk.Done → break
    
    // Assemble final response
    finalText := fullText.String()
    session.Messages = append sensei response
    statusCh <- "done:" + finalText
    return
  
  else:
    // existing non-streaming path (unchanged)
```

### Channel buffer size

The `statusChannelBufSize` (currently 10) may be insufficient when streaming produces many small chunks. Increase to accommodate streaming — suggested value: 50.

### Status channel contract

The status channel contract remains unchanged:
- String messages for status/progress updates
- `"done:..."` prefix for final response
- `"error:..."` prefix for errors

Stream chunks are additional string messages in the channel — existing consumers (TUI) that drain the channel without pattern matching on chunk text will continue to work. Consumers that want progressive display can recognize chunk messages as any message that is NOT "Pensando...", NOT "Ejecutando...", NOT starting with "Métricas:", NOT starting with "done:", and NOT starting with "error:".

### Files affected

- `backend/internal/core/services/sensei_service.go` — modify `runAgentLoop` for streaming path, increase buffer size
- `backend/internal/core/services/sensei_service_test.go` — add `SendMessageStream` to mock provider, add streaming tests

### Requirements

- **REQ-SAL-01**: Agent loop uses streaming when `tools == nil` AND provider supports streaming
- **REQ-SAL-02**: Stream chunks forwarded via status channel (as string messages)
- **REQ-SAL-03**: Final response correctly assembled from concatenated stream chunks
- **REQ-SAL-04**: Tool-enabled requests (`tools != nil`) use non-streaming path (unchanged)
- **REQ-SAL-05**: Existing status channel consumers (TUI) receive stream chunks without modification — the channel type and message format are unchanged

### Scenarios

**Scenario SAL-01: Text-only request uses streaming**
```
GIVEN a SenseiService with a streaming-capable provider
  AND tools == nil (no tool intent detected in user message)
  AND user says "holi"
WHEN runAgentLoop executes
THEN SendMessageStream is called (not SendMessage)
  AND stream chunks appear in the status channel
  AND the final "done:" message contains the full assembled text
```

**Scenario SAL-02: Tool-enabled request uses non-streaming**
```
GIVEN a SenseiService with a streaming-capable provider
  AND tools != nil (tool intent detected, e.g., "creá un archivo")
WHEN runAgentLoop executes
THEN SendMessage (non-streaming) is called
  AND SendMessageStream is NOT called
  AND behavior is identical to before
```

**Scenario SAL-03: Streamed response assembly**
```
GIVEN streaming produces chunks: ["¡", "Bue", "nas! ", "¿En ", "qué ", "te ayudo?"]
WHEN runAgentLoop collects chunks
THEN the final response is "¡Buenas! ¿En qué te ayudo?"
  AND the sensei response in the session is the full string
```

**Scenario SAL-04: Status channel contains chunks**
```
GIVEN streaming produces 3 text chunks
WHEN consuming the status channel
THEN the channel contains 3 additional messages (the chunks)
  PLUS the standard pause and done messages
```

---

## 4. Cross-cutting Invariants

- **INV-01**: System prompt in Spanish Rioplatense preserved on all model tiers (fast, flash, heavy)
- **INV-02**: All existing tests pass without modification — changes are additive or transparent;
  mock provider in `sensei_service_test.go` needs `SendMessageStream` added (compilation fix only)
- **INV-03**: `SenseiProvider.SendMessage` signature unchanged — streaming is additive
- **INV-04**: `maxAgentRounds` (5) unchanged
- **INV-05**: Timeout configuration unchanged (same env vars: `GODOJO_SENSEI_TIMEOUT_SECONDS`, `GODOJO_SENSEI_TOOL_TIMEOUT_SECONDS`)
- **INV-06**: Graceful degradation without API key preserved — all paths check `apiKey == ""` first
- **INV-07**: Non-streaming response parsing (thought filtering, empty response handling) identical for `GenerateContentResponse` consumed via both streaming and non-streaming paths
