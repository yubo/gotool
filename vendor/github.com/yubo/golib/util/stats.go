package util

import (
	"sync/atomic"
)

type Stats struct {
	keys   []string
	values []uint64
}

func NewStats(keys []string, values []uint64) *Stats {
	if len(keys) != len(values) {
		panic("len(keys) != len(values)")
	}

	return &Stats{
		values: values,
		keys:   keys,
	}
}

func (p *Stats) Dec(idx, n int) {
	atomic.AddUint64(&p.values[idx], ^uint64(n-1))
}

func (p *Stats) Inc(idx, n int) {
	atomic.AddUint64(&p.values[idx], uint64(n))
}

func (p *Stats) Set(idx, n int) {
	atomic.StoreUint64(&p.values[idx], uint64(n))
}

func (p *Stats) Get(idx int) uint64 {
	return atomic.LoadUint64(&p.values[idx])
}

func (p *Stats) GetKeys() []string {
	return p.keys
}

func (p *Stats) GetValues() []uint64 {
	ret := make([]uint64, len(p.values))
	for i := 0; i < len(p.values); i++ {
		ret[i] = atomic.LoadUint64(&p.values[i])
	}
	return ret
}

func (p *Stats) GetKvs() map[string]uint64 {
	ret := map[string]uint64{}
	for i := 0; i < len(p.values); i++ {
		ret[p.keys[i]] = atomic.LoadUint64(&p.values[i])
	}
	return ret
}
