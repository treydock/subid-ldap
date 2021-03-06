// Copyright 2021 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/treydock/subid-ldap/internal/config"
)

const (
	metricsNamespace = "subid_ldap"
	metricsPath      = "/metrics"
)

var (
	metricBuildInfo = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "build_info",
		Help:      "Build information",
		ConstLabels: prometheus.Labels{
			"version":   version.Version,
			"revision":  version.Revision,
			"branch":    version.Branch,
			"builddate": version.BuildDate,
			"goversion": version.GoVersion,
		},
	})
	MetricError = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "error",
		Help:      "Indicates an error was encountered",
	})
	MetricDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "run_duration_seconds",
		Help:      "Last runtime duration in seconds",
	})
	MetricLastRun = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "last_run_timestamp_seconds",
		Help:      "Last timestamp of execution",
	})
	MetricSubIDTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "subid_total",
		Help:      "Total number of subid entries",
	})
	MetricSubIDAdded = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "subid_added",
		Help:      "Number of subid entries added",
	})
	MetricSubIDRemoved = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "subid_removed",
		Help:      "Number of subid entries removed",
	})
)

func init() {
	ResetMetrics()
}

func ResetMetrics() {
	metricBuildInfo.Set(1)
	MetricError.Set(0)
	MetricSubIDTotal.Set(0)
	MetricSubIDAdded.Set(0)
	MetricSubIDRemoved.Set(0)
}

func MetricGathers(processMetrics bool) prometheus.Gatherers {
	registry := prometheus.NewRegistry()
	registry.MustRegister(metricBuildInfo)
	registry.MustRegister(MetricError)
	registry.MustRegister(MetricDuration)
	registry.MustRegister(MetricLastRun)
	registry.MustRegister(MetricSubIDTotal)
	registry.MustRegister(MetricSubIDAdded)
	registry.MustRegister(MetricSubIDRemoved)
	gatherers := prometheus.Gatherers{registry}
	if processMetrics {
		gatherers = append(gatherers, prometheus.DefaultGatherer)
	}
	return gatherers
}

func MetricsServer(listenAddress string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	             <head><title>` + config.AppName + `</title></head>
	             <body>
	             <h1>` + config.AppName + `</h1>
	             <p><a href='` + metricsPath + `'>Metrics</a></p>
	             </body>
	             </html>`))
	})
	http.Handle(metricsPath, promhttp.HandlerFor(MetricGathers(true), promhttp.HandlerOpts{}))
	return http.ListenAndServe(listenAddress, nil)
}

func MetricsWrite(path string, gatherers prometheus.Gatherers, logger log.Logger) {
	err := prometheus.WriteToTextfile(path, gatherers)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to write metrics file", "err", err)
	}
}

func Duration() func() {
	start := time.Now()
	return func() {
		MetricDuration.Set(time.Since(start).Seconds())
	}
}

func Error() func(*error) {
	return func(err *error) {
		if *err != nil {
			MetricError.Set(1)
		}
	}
}
