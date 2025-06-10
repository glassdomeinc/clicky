package chotel

import (
	"context"
	"database/sql"
	"runtime"
	"strings"

	"github.com/glassdomeinc/clicky/ch"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumName = "github.com/glassdomeinc/clicky/chotel"

var tracer = otel.Tracer("go-clickhouse")

type QueryHook struct{}

var _ ch.QueryHook = (*QueryHook)(nil)

func NewQueryHook() *QueryHook {
	return &QueryHook{}
}

func (h *QueryHook) BeforeQuery(
	ctx context.Context, evt *ch.QueryEvent,
) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	ctx, _ = tracer.Start(ctx, "", trace.WithSpanKind(trace.SpanKindClient))
	return ctx
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *ch.QueryEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	defer span.End()

	operation := event.Operation()
	fn, file, line := funcFileLine("go-clickhouse")
	span.SetName(operation)

	attrs := []attribute.KeyValue{
		semconv.CodeFunctionNameKey.String(fn),
		semconv.CodeFilepathKey.String(file),
		semconv.CodeLineNumberKey.Int(line),
		semconv.DBSystemNameKey.String("clickhouse"),
		semconv.DBOperationNameKey.String(operation),
		attribute.Key("db.statement").String(event.Query),
	}

	if event.IQuery != nil {
		if tableName := event.IQuery.GetTableName(); tableName != "" {
			attrs = append(attrs, attribute.Key("db.sql.table").String(tableName))
		}
	}

	span.SetAttributes(attrs...)

	switch event.Err {
	case nil, sql.ErrNoRows:
	default:
		span.SetStatus(codes.Error, "")
		span.RecordError(event.Err)
	}

	if event.Result != nil {
		numRow, err := event.Result.RowsAffected()
		if err == nil {
			span.SetAttributes(attribute.Int64("db.rows_affected", numRow))
		}
	}
}

func funcFileLine(pkg string) (string, string, int) {
	const depth = 16
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	ff := runtime.CallersFrames(pcs[:n])

	var fn, file string
	var line int
	for {
		f, ok := ff.Next()
		if !ok {
			break
		}
		fn, file, line = f.Function, f.File, f.Line
		if !strings.Contains(fn, pkg) {
			break
		}
	}

	if ind := strings.LastIndexByte(fn, '/'); ind != -1 {
		fn = fn[ind+1:]
	}

	return fn, file, line
}

type config struct {
	meterProvider  metric.MeterProvider
	meter          metric.Meter
	attrs          []attribute.KeyValue
	queryFormatter func(query string) string
}

type Option func(c *config)

// WithAttributes configures attributes that are used to create a span.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return func(c *config) {
		c.attrs = append(c.attrs, attrs...)
	}
}

// WithDBSystem configures a db.system attribute. You should prefer using
// WithAttributes and semconv, for example, `otelsql.WithAttributes(semconv.DBSystemSqlite)`.
func WithDBSystem(system string) Option {
	return func(c *config) {
		c.attrs = append(c.attrs, semconv.DBSystemNameKey.String(system))
	}
}

// WithDBName configures a db.name attribute.
func WithDBName(name string) Option {
	return func(c *config) {
		c.attrs = append(c.attrs, attribute.Key("db.name").String(name))
	}
}

// WithMeterProvider configures a metric.Meter used to create instruments.
func WithMeterProvider(meterProvider metric.MeterProvider) Option {
	return func(c *config) {
		c.meterProvider = meterProvider
	}
}

// WithQueryFormatter configures a query formatter
func WithQueryFormatter(queryFormatter func(query string) string) Option {
	return func(c *config) {
		c.queryFormatter = queryFormatter
	}
}

func newConfig(opts []Option) *config {
	c := &config{
		meterProvider: otel.GetMeterProvider(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ReportDBStatsMetrics reports DBStats metrics using OpenTelemetry Metrics API.
func ReportDBStatsMetrics(db *ch.DB, opts ...Option) {
	cfg := newConfig(opts)

	if cfg.meter == nil {
		cfg.meter = cfg.meterProvider.Meter(instrumName)
	}

	meter := cfg.meter
	labels := cfg.attrs

	totalPoolConns, _ := meter.Int64ObservableGauge(
		"go.ch.pool_connections_total",
		metric.WithDescription("Number of total connections in the pool"),
	)
	idlePoolConns, _ := meter.Int64ObservableGauge(
		"go.ch.pool_connections_idle",
		metric.WithDescription("Number of idle connections in the pool"),
	)

	idlePoolStale, _ := meter.Int64ObservableGauge(
		"go.ch.pool_connections_stale",
		metric.WithDescription("Number of stale connections removed from the pool"),
	)

	if _, err := meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			stats := db.Stats()
			o.ObserveInt64(totalPoolConns, int64(stats.PoolStats.TotalConns), metric.WithAttributes(labels...))
			o.ObserveInt64(idlePoolConns, int64(stats.PoolStats.IdleConns), metric.WithAttributes(labels...))
			o.ObserveInt64(idlePoolStale, int64(stats.PoolStats.StaleConns), metric.WithAttributes(labels...))

			return nil
		},
		totalPoolConns,
		idlePoolConns,
		idlePoolStale,
	); err != nil {
		panic(err)
	}
}
