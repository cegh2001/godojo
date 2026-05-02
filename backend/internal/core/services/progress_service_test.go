package services_test

import (
	"context"
	"errors"
	"testing"

	"godojo/internal/core/domain"
	"godojo/internal/core/services"
)

// mockProgressStore implements ports.ProgressStore for testing.
type mockProgressStore struct {
	data      map[string]*domain.Progress
	loadErr   error
	saveErr   error
}

func newMockProgressStore() *mockProgressStore {
	return &mockProgressStore{
		data: make(map[string]*domain.Progress),
	}
}

func (m *mockProgressStore) Load(ctx context.Context) (map[string]*domain.Progress, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	// Return a copy to prevent mutation
	result := make(map[string]*domain.Progress, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result, nil
}

func (m *mockProgressStore) Save(ctx context.Context, progress map[string]*domain.Progress) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	// Copy into internal store
	m.data = make(map[string]*domain.Progress, len(progress))
	for k, v := range progress {
		m.data[k] = v
	}
	return nil
}

func (m *mockProgressStore) GetBySlug(ctx context.Context, slug string) (*domain.Progress, error) {
	p, ok := m.data[slug]
	if !ok {
		return nil, errors.New("progress not found")
	}
	return p, nil
}

func TestProgressService_MarkStarted(t *testing.T) {
	store := newMockProgressStore()
	svc := services.NewProgressService(store)

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "mark started valid slug", slug: "ex-1", wantErr: false},
		{name: "mark started empty slug", slug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.MarkStarted(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the progress was saved with in_progress status
			p, err := svc.GetProgress(tt.slug)
			if err != nil {
				t.Errorf("GetProgress failed: %v", err)
				return
			}
			if p.Status != domain.ProgressInProgress {
				t.Errorf("status = %q, want %q", p.Status, domain.ProgressInProgress)
			}
			if p.Attempts != 0 {
				t.Errorf("attempts = %d, want 0", p.Attempts)
			}
		})
	}
}

func TestProgressService_MarkCompleted(t *testing.T) {
	store := newMockProgressStore()
	svc := services.NewProgressService(store)

	// Must start first
	err := svc.MarkStarted("ex-1")
	if err != nil {
		t.Fatalf("MarkStarted failed: %v", err)
	}

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "mark completed after started", slug: "ex-1", wantErr: false},
		{name: "mark completed non-existent", slug: "no-existe", wantErr: true},
		{name: "mark completed empty slug", slug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.MarkCompleted(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			p, err := svc.GetProgress(tt.slug)
			if err != nil {
				t.Errorf("GetProgress failed: %v", err)
				return
			}
			if p.Status != domain.ProgressCompleted {
				t.Errorf("status = %q, want %q", p.Status, domain.ProgressCompleted)
			}
			if p.Attempts != 1 {
				t.Errorf("attempts = %d, want 1", p.Attempts)
			}
			if p.CompletedAt == nil {
				t.Error("completedAt should not be nil")
			}
		})
	}
}

func TestProgressService_MarkSkipped(t *testing.T) {
	store := newMockProgressStore()
	svc := services.NewProgressService(store)

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "mark skipped valid slug", slug: "ex-2", wantErr: false},
		{name: "mark skipped empty slug", slug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.MarkSkipped(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			p, err := svc.GetProgress(tt.slug)
			if err != nil {
				t.Errorf("GetProgress failed: %v", err)
				return
			}
			if p.Status != domain.ProgressSkipped {
				t.Errorf("status = %q, want %q", p.Status, domain.ProgressSkipped)
			}
		})
	}
}

func TestProgressService_GetCompletionPercent(t *testing.T) {
	store := newMockProgressStore()
	svc := services.NewProgressService(store)

	tests := []struct {
		name      string
		prepFn    func()
		topicSlug string
		wantPct   float64
		wantErr   bool
	}{
		{
			name: "empty topic returns 0%",
			prepFn: func() {},
			topicSlug: "test-topic",
			wantPct: 0,
			wantErr: false,
		},
		{
			name: "50% completed",
			prepFn: func() {
				// Setup: 2 exercises, 1 completed, 1 not_started
				// MarkCompleted adds exercises to "test-topic" implicitly
				_ = svc.MarkStarted("ex-a")
				_ = svc.MarkCompleted("ex-a")
				_ = svc.MarkStarted("ex-b")
			},
			topicSlug: "test-topic",
			wantPct:   50,
			wantErr:   false,
		},
		{
			name: "100% completed",
			prepFn: func() {
				_ = svc.MarkStarted("ex-c")
				_ = svc.MarkCompleted("ex-c")
			},
			topicSlug: "test-topic",
			wantPct:   100,
			wantErr:   false,
		},
		{
			name:      "empty topic slug",
			prepFn:    func() {},
			topicSlug: "",
			wantPct:   0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset store for each test case
			store.data = make(map[string]*domain.Progress)
			svc = services.NewProgressService(store)

			tt.prepFn()

			pct, err := svc.GetCompletionPercent(tt.topicSlug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got pct=%v", pct)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if pct != tt.wantPct {
				t.Errorf("completion = %v%%, want %v%%", pct, tt.wantPct)
			}
		})
	}
}

func TestProgressService_GetAllProgress(t *testing.T) {
	store := newMockProgressStore()
	svc := services.NewProgressService(store)

	// Empty initially
	all, err := svc.GetAllProgress()
	if err != nil {
		t.Fatalf("GetAllProgress failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 entries, got %d", len(all))
	}

	// Add some progress
	_ = svc.MarkStarted("ex-1")
	_ = svc.MarkStarted("ex-2")
	_ = svc.MarkCompleted("ex-1")

	all, err = svc.GetAllProgress()
	if err != nil {
		t.Fatalf("GetAllProgress failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all))
	}
}
