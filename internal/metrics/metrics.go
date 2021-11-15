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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

const (
	metricsNamespace = "subid_ldap"
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
	metricBuildInfo.Set(1)
	MetricError.Set(0)
	MetricSubIDTotal.Set(0)
	MetricSubIDAdded.Set(0)
	MetricSubIDRemoved.Set(0)
}

func MetricGathers() prometheus.Gatherers {
	registry := prometheus.NewRegistry()
	registry.MustRegister(metricBuildInfo)
	registry.MustRegister(MetricError)
	registry.MustRegister(MetricDuration)
	registry.MustRegister(MetricLastRun)
	registry.MustRegister(MetricSubIDTotal)
	registry.MustRegister(MetricSubIDAdded)
	registry.MustRegister(MetricSubIDRemoved)
	return prometheus.Gatherers{prometheus.DefaultGatherer, registry}
}

func MetricsWrite(path string, gatherers prometheus.Gatherers) error {
	err := prometheus.WriteToTextfile(path, gatherers)
	return err
}

func Duration() func() {
	start := time.Now()
	return func() {
		MetricDuration.Set(time.Since(start).Seconds())
	}
}
