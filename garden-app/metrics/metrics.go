package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	collectors []prometheus.Collector
}

func New(collectors ...prometheus.Collector) *Metrics {
	return &Metrics{collectors: collectors}
}

func (m *Metrics) AddCollector(c prometheus.Collector) {
	m.collectors = append(m.collectors, c)
}

func (m *Metrics) Register() {
	prometheus.MustRegister(m.collectors...)
}

func (m *Metrics) Unregister() {
	for _, c := range m.collectors {
		prometheus.Unregister(c)
	}
}
