# gin-prometheus
[![](https://godoc.org/github.com/wei840222/gin-prometheus?status.svg)](https://godoc.org/github.com/wei840222/gin-prometheus)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gin Web Framework Prometheus metrics exporter with exemplar

## Installation

`$ go get github.com/wei840222/gin-prometheus`

## Usage

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginprom "github.com/wei840222/gin-prometheus"
)

func main() {
	r := gin.New()

	p := ginprom.NewPrometheus("gin")
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Hello world!")
	})

	r.Run()
}
```

See the [example.go file](https://github.com/wei840222/gin-prometheus/blob/master/example/example.go)

## Preserving a low cardinality for the request counter

The request counter (`requests_total`) has a `url` label which,
although desirable, can become problematic in cases where your
application uses templated routes expecting a great number of
variations, as Prometheus explicitly recommends against metrics having
high cardinality dimensions:

https://prometheus.io/docs/practices/naming/#labels

If you have for instance a `/customer/:name` templated route and you
don't want to generate a time series for every possible customer name,
you could supply this mapping function to the middleware:

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginprom "github.com/wei840222/gin-prometheus"
)

func main() {
	r := gin.New()

	p := ginprom.NewPrometheus("gin")

	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := c.Request.URL.Path
		for _, p := range c.Params {
			if p.Key == "name" {
				url = strings.Replace(url, p.Value, ":name", 1)
				break
			}
		}
		return url
	}

	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Hello world!")
	})

	r.Run()
}
```

which would map `/customer/alice` and `/customer/bob` to their
template `/customer/:name`, and thus preserve a low cardinality for
our metrics.

## Using with OpenTelemetry Tracer and Meter
```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	ginprom "github.com/wei840222/gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

var ginOtelLogFormatter = func(param gin.LogFormatterParams) string {
	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}

	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v traceID=%s\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		trace.SpanContextFromContext(param.Request.Context()).TraceID(),
		param.ErrorMessage,
	)
}

func main() {
	mExp := otelprom.New()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(mExp))
	defer provider.Shutdown(context.Background())

	tExp, err := otlptrace.New(context.Background(), otlptracegrpc.NewClient())
	if err != nil {
		panic(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(tExp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("myservice"),
			semconv.ServiceVersionKey.String("0.0.1"),
		)),
	)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	p := ginprom.NewPrometheus("gin").SetEnableExemplar(true).SetOtelPromExporter(&mExp)
	p.SetListenAddress(":2222").SetMetricsPath(nil)

	e := gin.New()
	e.Use(otelgin.Middleware("gin", otelgin.WithTracerProvider(tp)), p.HandlerFunc(), gin.LoggerWithFormatter(ginOtelLogFormatter), gin.Recovery())

	e.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Hello world!")
	})
	e.GET("/panic", func(c *gin.Context) {
		panic("oh no")
	})

	e.Run()
}
```