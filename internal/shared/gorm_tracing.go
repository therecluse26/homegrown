package shared

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	dbTracerName  = "homegrown-academy/db"
	spanKey       = "otel:span"
)

// GormTracingPlugin is a GORM plugin that creates an OpenTelemetry span for every
// database operation (query, create, update, delete, raw SQL). Spans are children
// of whatever span is in the request context, giving full query-level visibility
// inside HTTP and background-job traces.
//
// Use: db.Use(shared.GormTracingPlugin{})
type GormTracingPlugin struct{}

func (GormTracingPlugin) Name() string { return "otel:tracing" }

func (p GormTracingPlugin) Initialize(db *gorm.DB) error {
	ops := []struct {
		before, after string
		name          string
	}{
		{"gorm:query", "gorm:after_query", "db.query"},
		{"gorm:create", "gorm:after_create", "db.create"},
		{"gorm:update", "gorm:after_update", "db.update"},
		{"gorm:delete", "gorm:after_delete", "db.delete"},
		{"gorm:row", "gorm:after_row", "db.raw"},
	}
	for _, op := range ops {
		if err := db.Callback().Query().Before(op.before).Register("otel:before_"+op.name, before(op.name)); err != nil {
			return fmt.Errorf("gorm otel: register before %s: %w", op.name, err)
		}
		if err := db.Callback().Query().After(op.after).Register("otel:after_"+op.name, after()); err != nil {
			return fmt.Errorf("gorm otel: register after %s: %w", op.name, err)
		}
	}
	return nil
}

func before(opName string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}
		tracer := otel.GetTracerProvider().Tracer(dbTracerName)
		_, span := tracer.Start(db.Statement.Context, opName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("db.system", "postgresql"),
				attribute.String("db.name", db.Name()),
			),
		)
		db.Statement.Settings.Store(spanKey, span)
	}
}

func after() func(*gorm.DB) {
	return func(db *gorm.DB) {
		val, ok := db.Statement.Settings.Load(spanKey)
		if !ok {
			return
		}
		span, ok := val.(trace.Span)
		if !ok {
			return
		}
		defer span.End()

		if db.Statement != nil && db.Statement.SQL.String() != "" {
			span.SetAttributes(attribute.String("db.statement", db.Statement.SQL.String()))
		}
		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			span.SetStatus(codes.Error, db.Error.Error())
			span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", db.Error)))
		}
	}
}
