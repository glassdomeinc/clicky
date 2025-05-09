# ClickHouse client for Go

This is a fork from uptace

This ClickHouse client uses native protocol to communicate with ClickHouse server. This is not a database/sql driver, but the API is compatible.

Main features are:

- ClickHouse native protocol support and efficient column-oriented design.
- API compatible with database/sql.
- [Bun](https://github.com/uptrace/bun/)-like query builder.
- [Selecting](https://clickhouse.uptrace.dev/guide/clickhouse-select.html) into scalars, structs,
  maps, slices of maps/structs/scalars.
- `Date`, `DateTime`, and `DateTime64`.
- `Array(T)` including nested arrays.
- Enums and `LowCardinality(String)`.
- `Nullable(T)` except `Nullable(Array(T))`.
- [Migrations](https://clickhouse.uptrace.dev/guide/clickhouse-migrations.html).
- [OpenTelemetry](https://clickhouse.uptrace.dev/guide/clickhouse-monitoring-performance.html)
  support.
- In production at [Uptrace](https://github.com/uptrace/uptrace)

Resources:

- [**Get started**](https://clickhouse.uptrace.dev/guide/getting-started.html)
- [Examples](https://github.com/glassdomeinc/clicky/tree/master/example)

## Installation

```shell
go get github.com/glassdomeinc/clicky@latest
```

## Example

A [basic](example/basic) example:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/glassdomeinc/clicky/ch"
	"github.com/glassdomeinc/clicky/chdebug"
)

type Model struct {
	ch.CHModel `ch:"partition:toYYYYMM(time)"`

	ID   uint64
	Text string    `ch:",lc"`
	Time time.Time `ch:",pk"`
}

func main() {
	ctx := context.Background()

	db := ch.Connect(ch.WithDatabase("test"))
	db.AddQueryHook(chdebug.NewQueryHook(chdebug.WithVerbose(true)))

	if err := db.Ping(ctx); err != nil {
		panic(err)
	}

	var num int
	if err := db.QueryRowContext(ctx, "SELECT 123").Scan(&num); err != nil {
		panic(err)
	}
	fmt.Println(num)

	if err := db.ResetModel(ctx, (*Model)(nil)); err != nil {
		panic(err)
	}

	src := &Model{ID: 1, Text: "hello", Time: time.Now()}
	if _, err := db.NewInsert().Model(src).Exec(ctx); err != nil {
		panic(err)
	}

	dest := new(Model)
	if err := db.NewSelect().Model(dest).Where("id = ?", src.ID).Limit(1).Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Println(dest)
}
```
