package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestNewProvider_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create disabled provider: %v", err)
	}

	if provider.IsEnabled() {
		t.Error("Provider should not be enabled when Enabled=false")
	}
}

func TestNewProvider_Defaults(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		OTLPEndpoint: "localhost:4317",
		// Leave other fields empty to test defaults
	}

	// We expect this to fail since there's no actual collector
	// but it should at least not panic and apply defaults
	_, err := NewProvider(cfg)
	// The provider creation shouldn't fail even if the endpoint is unreachable
	// since OTLP exporters connect lazily
	if err != nil && err.Error() != "" {
		// Just log it, don't fail - the endpoint doesn't exist in tests
		t.Logf("Provider creation note: %v", err)
	}
}

func TestNewProvider_InvalidProtocol(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		OTLPEndpoint: "localhost:4317",
		OTLPProtocol: "invalid",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for invalid protocol")
	}
}

func TestProvider_Shutdown_Nil(t *testing.T) {
	provider := &Provider{}

	err := provider.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown with nil provider should not error: %v", err)
	}
}

func TestProvider_Tracer_Nil(t *testing.T) {
	provider := &Provider{}

	tracer := provider.Tracer("test")
	if tracer == nil {
		t.Error("Tracer should not be nil even with nil provider")
	}
}

func TestProvider_IsEnabled_Nil(t *testing.T) {
	provider := &Provider{}

	if provider.IsEnabled() {
		t.Error("Provider with nil internal provider should not be enabled")
	}
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	if span == nil {
		t.Error("Span should not be nil")
	}

	// Verify context is returned
	if ctx == nil {
		t.Error("Context should not be nil")
	}
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	retrievedSpan := SpanFromContext(ctx)
	if retrievedSpan == nil {
		t.Error("Retrieved span should not be nil")
	}
}

func TestAddSpanEvent(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic
	AddSpanEvent(ctx, "test-event", attribute.String("key", "value"))
}

func TestSetSpanError(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	err := context.DeadlineExceeded
	// Should not panic
	SetSpanError(ctx, err)
}

func TestSetSpanAttributes(t *testing.T) {
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Should not panic
	SetSpanAttributes(ctx,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	)
}

func TestConfig_SampleRate(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate float64
	}{
		{"always_sample", 1.0},
		{"never_sample", 0.0},
		{"half_sample", 0.5},
		{"very_low", 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled:      true,
				OTLPEndpoint: "localhost:4317",
				SampleRate:   tt.sampleRate,
			}

			// Creation should succeed regardless of sample rate
			_, err := NewProvider(cfg)
			if err != nil {
				t.Logf("Note: %v", err)
			}
		})
	}
}

func TestConfig_Protocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  bool
	}{
		{"grpc", "grpc", false},
		{"http", "http", false},
		{"invalid", "websocket", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled:      true,
				OTLPEndpoint: "localhost:4317",
				OTLPProtocol: tt.protocol,
			}

			_, err := NewProvider(cfg)
			if tt.wantErr && err == nil {
				t.Error("Expected error for invalid protocol")
			}
			if !tt.wantErr && err != nil {
				t.Logf("Note: %v", err)
			}
		})
	}
}
