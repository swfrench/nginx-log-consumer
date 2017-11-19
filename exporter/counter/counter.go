package counter

import (
	"time"
)

// CounterMetricT provides an interface implemented by all cumulative counter
// metrics. Can be used, for example, to implement mock counters for tests.
type CounterMetricT interface {
	Create() error
	Increment(map[string]int64) error
	ResetTime() time.Time
}
