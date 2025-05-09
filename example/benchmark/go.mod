module github.com/glassdomeinc/clicky/ch/internal/bench

go 1.18

replace github.com/glassdomeinc/clicky => ../..

replace github.com/glassdomeinc/clicky/chdebug => ../../chdebug

require github.com/glassdomeinc/clicky v0.3.1

require (
	github.com/codemodus/kace v0.5.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1 // indirect
)
