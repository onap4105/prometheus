// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, softwar
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package promql

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/util/testutil"
)

// A Benchmark holds context for running a unit test as a benchmark.
type Benchmark struct {
	b         *testing.B
	t         *Test
	iterCount int
}

// NewBenchmark returns an initialized empty Benchmark.
func NewBenchmark(b *testing.B, input string) *Benchmark {
	t, err := NewTest(b, input)
	if err != nil {
		b.Fatalf("Unable to run benchmark: %s", err)
	}
	return &Benchmark{
		b: b,
		t: t,
	}
}

// Run runs the benchmark.
func (b *Benchmark) Run() {
	defer b.t.Close()
	b.b.ReportAllocs()
	b.b.ResetTimer()
	for i := 0; i < b.b.N; i++ {
		if err := b.t.RunAsBenchmark(b); err != nil {
			b.b.Error(err)
		}
		b.iterCount++
	}
}

func BenchmarkRangeQuery(b *testing.B) {
	storage := testutil.NewStorage(b)
	defer storage.Close()
	engine := NewEngine(nil, nil, 10, 10*time.Second)

	a, err := storage.Appender()
	if err != nil {
		b.Fatal(err)
	}
	for s := 0; s < 10000; s += 1 {
		ts := int64(s * 10000) // 10s interval.
		metric := labels.FromStrings("__name__", "a_one")
		a.Add(metric, ts, float64(s))
		metric = labels.FromStrings("__name__", "b_one")
		a.Add(metric, ts, float64(s))

		for i := 0; i < 10; i++ {
			metric = labels.FromStrings("__name__", "a_ten", "l", strconv.Itoa(i))
			a.Add(metric, ts, float64(s))
			metric = labels.FromStrings("__name__", "b_ten", "l", strconv.Itoa(i))
			a.Add(metric, ts, float64(s))
		}
		for i := 0; i < 100; i++ {
			metric = labels.FromStrings("__name__", "a_hundred", "l", strconv.Itoa(i))
			a.Add(metric, ts, float64(s))
			metric = labels.FromStrings("__name__", "b_hundred", "l", strconv.Itoa(i))
			a.Add(metric, ts, float64(s))
		}
	}
	if err := a.Commit(); err != nil {
		b.Fatal(err)
	}

	cases := []struct {
		expr     string
		interval time.Duration
		steps    int64
	}{
		{
			expr:     "rate(a_one[1m])",
			interval: time.Second * 10,
			steps:    1,
		},
		{
			expr:     "rate(a_one[1m])",
			interval: time.Second * 10,
			steps:    10,
		},
		{
			expr:     "rate(a_one[1m])",
			interval: time.Second * 10,
			steps:    1000,
		},
		{
			expr:     "rate(a_ten[1m])",
			interval: time.Second * 10,
			steps:    1,
		},
		{
			expr:     "rate(a_ten[1m])",
			interval: time.Second * 10,
			steps:    10,
		},
		{
			expr:     "rate(a_ten[1m])",
			interval: time.Second * 10,
			steps:    1000,
		},
		{
			expr:     "rate(a_hundred[1m])",
			interval: time.Second * 10,
			steps:    1,
		},
		{
			expr:     "rate(a_hundred[1m])",
			interval: time.Second * 10,
			steps:    100,
		},
		{
			expr:     "rate(a_hundred[1m])",
			interval: time.Second * 10,
			steps:    1000,
		},
	}
	for _, c := range cases {
		name := fmt.Sprintf("expr=%s,interval=%s,steps=%d", c.expr, c.interval, c.steps)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				end := c.steps*int64(c.interval.Seconds()) - 1
				qry, err := engine.NewRangeQuery(storage, c.expr, time.Unix(0, 0), time.Unix(end, 0), c.interval)
				if err != nil {
					b.Fatal(err)
				}
				res := qry.Exec(context.Background())
				if res.Err != nil {
					b.Fatal(res.Err)
				}
			}
		})
	}
}
