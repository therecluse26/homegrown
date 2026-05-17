package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "homegrown-academy/http"

// Tracing returns an Echo middleware that:
//  1. Extracts incoming W3C traceparent / tracestate headers so inbound requests
//     from load balancers, gateways, or other services continue an existing trace.
//  2. Starts a new server span for every request, named after the matched route
//     template (e.g. "GET /v1/families/:id") to avoid high-cardinality span names.
//  3. Records HTTP semantic-convention attributes (method, route, status code).
//  4. Marks the span as error only on 5xx — 4xx are client faults, not server faults.
//  5. Injects the outgoing traceparent header so downstream services can link spans.
func Tracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			propagator := otel.GetTextMapPropagator()

			// Extract parent trace context from incoming headers (W3C traceparent).
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			// Use the matched route template to avoid per-URL cardinality explosion.
			// At middleware time the route may not be matched yet; we update after next().
			tracer := otel.GetTracerProvider().Tracer(tracerName)
			spanName := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.ServerAddress(req.Host),
				),
			)
			defer span.End()

			// Propagate the updated context so handlers can create child spans.
			c.SetRequest(req.WithContext(ctx))

			// Inject outgoing trace context into response headers so downstream
			// services (and browser-side RUM agents) can link their spans.
			propagator.Inject(ctx, propagation.HeaderCarrier(c.Response().Header()))

			err := next(c)

			// Update span name with the matched route template now that Echo has routed.
			if route := c.Path(); route != "" {
				span.SetName(fmt.Sprintf("%s %s", req.Method, route))
				span.SetAttributes(semconv.HTTPRouteKey.String(route))
			}

			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			span.SetAttributes(semconv.HTTPResponseStatusCode(statusCode))

			// Only mark server errors — 4xx are client errors, not our fault.
			if statusCode >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			}

			if err != nil {
				span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
			}

			return err
		}
	}
}
