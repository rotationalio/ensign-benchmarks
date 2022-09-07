package stats

import (
	"encoding/json"
	"sync"
	"time"
)

// Latencies keeps track of a distribution of durations, e.g. to benchmark the
// performance or timing of an operation. It returns descriptive statistics
// as durations so that they can be read as timings. Latencies works in an
// online fashion similar to the Statistics object, but works on
// time.Duration samples instead of floats. Instead of minimum and maximum
// values it returns the fastest and slowest times.
//
// The primary entry point to the object is via the Update method, where one
// or more time.Durations can be passed. This object has unexported fields
// because it is thread-safe (via a sync.RWMutex). All properties must be
// accessed from read-locked access methods.
type Latencies struct {
	sync.RWMutex
	Statistics
	timeouts uint64        // the number of 0 durations (null durations) or timeouts
	duration time.Duration // externally set duration of the benchmark
}

// Update the latencies with a duration or durations (thread-safe). If a
// duration of 0 is passed, then it is interpreted as a timeout -- e.g. a
// maximal duration bound had been reached. Timeouts are recorded in a
// separate counter and can be used to express failure measures.
func (s *Latencies) Update(durations ...time.Duration) {
	s.Lock()
	defer s.Unlock()

	for _, duration := range durations {
		// Record any timeouts in the benchmark
		if duration == 0 {
			s.timeouts++
			continue
		}

		s.Statistics.Update(duration.Seconds())
	}
}

// SetDuration allows an external setting of the duration. This is especially
// useful in the case where multiple threads are updating the latencies and
// the internal measurement of total time might double count concurrent
// accesses. In fact it is strongly recommended that this method is called
// from the external measurer after all updating is complete.
func (s *Latencies) SetDuration(duration time.Duration) {
	s.Lock()
	defer s.Unlock()
	s.duration = duration
}

// Throughput returns the number of samples per second, measured as the
// inverse mean: number of samples divided by the total duration in seconds.
// The duration is computed in two ways:
//
//   - if SetDuration is called, that duration is used
//   - otherwise, the total number of observed seconds is used
//
// This metric does not express a duration, so a float64 value is returned
// instead. If the duration or number of accesses is zero, 0.0 is returned.
func (s *Latencies) Throughput() float64 {
	s.RLock()
	defer s.RUnlock()
	return s.throughput()
}

func (s *Latencies) throughput() float64 {
	if s.samples > 0 && s.duration > 0 {
		return float64(s.Statistics.samples) / s.duration.Seconds()
	}

	if s.samples > 0 && s.total > 0 {
		return float64(s.Statistics.samples) / s.Statistics.total
	}
	return 0.0
}

// Total returns the total duration recorded across all samples.
func (s *Latencies) Total() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.ltotal()
}

// named ltotal to not conflict with the underlying total property.
func (s *Latencies) ltotal() time.Duration {
	return s.castSeconds(s.Statistics.total)
}

// Timeouts returns the number of timeouts recorded across all samples.
func (s *Latencies) Timeouts() uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.timeouts
}

// Mean returns the average for all durations expressed as float64 seconds
// and returns a time.Duration which is expressed in int64 nanoseconds. This
// can mean some loss in precision of the mean value, but also allows the
// caller to compute the mean in varying timescales. Since microseconds is a
// pretty fine granularity for timings, truncating the floating point of the
// nanosecond seems acceptable.
//
// If no durations have been recorded, a zero valued duration is returned.
func (s *Latencies) Mean() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.mean()
}

func (s *Latencies) mean() time.Duration {
	return s.castSeconds(s.Statistics.Mean())
}

// Variance computes the variability of samples and describes the distance of
// the distribution from the mean. This function returns a time.Duration,
// which can mean a loss in precision lower than the microsecond level. This
// is usually acceptable for most applications.
//
// If no more than 1 durations were recorded, returns a zero valued duration.
func (s *Latencies) Variance() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.variance()
}

func (s *Latencies) variance() time.Duration {
	return s.castSeconds(s.Statistics.Variance())
}

// StdDev returns the standard deviation of samples, the square root of the
// variance. This function returns a time.Duration which represents a loss in
// precision from int64 nanoseconds to float64 seconds.
//
// If no more than 1 durations were recorded, returns a zero valued duration.
func (s *Latencies) StdDev() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.stddev()
}

func (s *Latencies) stddev() time.Duration {
	return s.castSeconds(s.Statistics.StdDev())
}

// Slowest returns the maximum value of durations seen. If no durations have
// been added to the dataset, then this function returns a zero duration.
func (s *Latencies) Slowest() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.slowest()
}

func (s *Latencies) slowest() time.Duration {
	return s.castSeconds(s.Statistics.Maximum())
}

// Fastest returns the minimum value of durations seen. If no durations have
// been added to the dataset, then this function returns a zero duration.
func (s *Latencies) Fastest() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.fastest()
}

func (s *Latencies) fastest() time.Duration {
	return s.castSeconds(s.Statistics.Minimum())
}

// Range returns the difference between the slowest and fastest durations.
// If no samples have been added to the dataset, this function returns a zero
// duration. It will also return zero if the fastest and slowest durations
// are equal. E.g. in the case only one duration has been recorded or such
// that all durations have the same value.
func (s *Latencies) Range() time.Duration {
	s.RLock()
	defer s.RUnlock()
	return s.lrange()
}

// Named lrange so as to not conflict with the range keyword
func (s *Latencies) lrange() time.Duration {
	return s.castSeconds(s.Statistics.Range())
}

// Serializes the metric into a JSON map of named summary statistics.
func (s *Latencies) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	data := make(map[string]interface{})
	data["samples"] = s.samples
	data["total"] = s.ltotal().String()
	data["mean"] = s.mean().String()
	data["stddev"] = s.stddev().String()
	data["variance"] = s.variance().String()
	data["fastest"] = s.fastest().String()
	data["slowest"] = s.slowest().String()
	data["range"] = s.lrange().String()
	data["throughput"] = s.throughput()
	data["duration"] = s.duration.String()
	data["timeouts"] = s.timeouts
	return json.Marshal(data)
}

// Append another benchmark object to the current benchmark object,
// incrementing the distribution from the other object.
func (s *Latencies) Append(o *Latencies) {
	s.Statistics.Append(&o.Statistics)
	s.timeouts += o.timeouts
}

// Internal Helper Method to cast float64 seconds into a duration
func (s *Latencies) castSeconds(seconds float64) time.Duration {
	return time.Duration(float64(time.Second) * seconds)
}
