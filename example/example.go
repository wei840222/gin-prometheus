package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ginprom "github.com/wei840222/gin-prometheus"
)

func main() {
	r := gin.New()

	/*
		// Optional custom metrics list
		customMetrics := []*ginprom.Metric{
			{
				ID:          "1234",                // optional string
				Name:        "test_metric",         // required string
				Description: "Counter test metric", // required string
				Type:        "counter",             // required string
			},
			{
				ID:          "1235",                // Identifier
				Name:        "test_metric_2",       // Metric Name
				Description: "Summary test metric", // Help Description
				Type:        "summary",             // type associated with prometheus collector
			},
			// Type Options:
			//	counter, counter_vec, gauge, gauge_vec,
			//	histogram, histogram_vec, summary, summary_vec
		}
		p := ginprom.NewPrometheus("gin", customMetrics)
	*/

	p := ginprom.NewPrometheus("gin")

	p.Use(r)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Hello world!")
	})

	r.Run()
}
