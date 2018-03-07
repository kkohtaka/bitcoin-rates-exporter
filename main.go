// Copyright (C) 2018 Kazumasa Kohtaka <kkohtaka@gmail.com> All right reserved
// This file is available under the MIT license.

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace   string = "bitcoin"
	apiEndpoint string = "https://blockchain.info/ticker"
)

var (
	address = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
)

type exporter struct {
	client *http.Client
	mutex  sync.RWMutex

	up           prometheus.Gauge
	totalScrapes prometheus.Counter
	exchangeRate *prometheus.GaugeVec
}

// Describe sends the descriptors of metrics
func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.totalScrapes.Describe(ch)
	e.exchangeRate.Describe(ch)
}

// Collect is called by the Prometheus registry when collecting metrics.
func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.scrape()

	e.up.Collect(ch)
	e.totalScrapes.Collect(ch)
	e.exchangeRate.Collect(ch)
}

func (e *exporter) scrape() {
	resp, err := e.client.Get(apiEndpoint)
	if err != nil {
		log.Printf("%+v\n", errors.Wrap(err, "send HTTP request"))
		e.up.Set(0)
		return
	}
	defer resp.Body.Close()

	var prices map[string]struct {
		LTP float64 `json:"last"`
		Ask float64 `json:"buy"`
		Bid float64 `json:"sell"`
	}
	err = json.NewDecoder(resp.Body).Decode(&prices)
	if err != nil {
		log.Printf("%+v\n", errors.Wrap(err, "decode HTTP response as JSON"))
		e.up.Set(0)
		return
	}

	e.totalScrapes.Inc()
	e.up.Set(1)

	for currency, price := range prices {
		e.exchangeRate.With(prometheus.Labels{
			"currency": currency,
			"class":    "ltp",
		}).Set(price.LTP)

		e.exchangeRate.With(prometheus.Labels{
			"currency": currency,
			"class":    "ask",
		}).Set(price.Ask)

		e.exchangeRate.With(prometheus.Labels{
			"currency": currency,
			"class":    "bid",
		}).Set(price.Bid)
	}
}

func newExporter() *exporter {
	e := exporter{
		client: &http.Client{},

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of Blockchain Exchange Rates API successful",
		}),

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_total_scrapes",
			Help:      "Current total Blockchain Exchange Rates API scrapes",
		}),

		exchangeRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "exchange_rate",
				Help:      "Exchange rate retrieved by Blockchain Exchange Rates API",
			},
			[]string{"currency", "class"},
		),
	}

	e.up.Set(0)
	e.totalScrapes.Add(0)

	return &e
}

func main() {
	flag.Parse()
	prometheus.MustRegister(newExporter())
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*address, nil))
}
