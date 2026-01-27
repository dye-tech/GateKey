package telemetry

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// GinMiddleware returns a Gin middleware that creates spans for HTTP requests.
// It extracts trace context from incoming requests and propagates it.
func GinMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)

	return func(c *gin.Context) {
		// Extract trace context from incoming request headers
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start a new span for this request
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLPath(c.Request.URL.Path),
				semconv.URLScheme(c.Request.URL.Scheme),
				semconv.UserAgentOriginal(c.Request.UserAgent()),
				semconv.ClientAddress(c.ClientIP()),
			),
		)
		defer span.End()

		// Store the span context in the request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record response attributes
		status := c.Writer.Status()
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(status),
			attribute.Int("http.response.size", c.Writer.Size()),
		)

		// Mark span as error if status >= 400
		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}
