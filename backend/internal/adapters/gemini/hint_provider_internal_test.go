package gemini

import "testing"

func TestNewHintProvider_UsesFastModelEnv(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv(senseiFastModelEnv, defaultHeavyModel)

	provider := NewHintProvider()

	client, ok := provider.client.(*realGeminiClient)
	if !ok {
		t.Fatalf("client type = %T", provider.client)
	}
	if client.model != defaultHeavyModel {
		t.Fatalf("model = %q, want %q", client.model, defaultHeavyModel)
	}
}