/*
 * Copyright 2017 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package ratelimits

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	RL_MAX_BITS = 8
)

type entry struct {
	ts []time.Time
	i  uint32
	sync.RWMutex
}

type RateLimits struct {
	members map[string]*entry
	hz      uint32 //  HZ
	bits    uint32 //  slices per cycle
	size    uint32
	mask    uint32
	offset  time.Duration
	sync.RWMutex
}

/* find first set bit */
func ffs(mask uint32) uint32 {
	var bit uint32

	if mask == 0 {
		return 0
	}
	for bit = 1; mask&1 == 0; bit++ {
		mask = mask >> 1
	}
	return bit
}

func New(hz, accuracy uint32) (*RateLimits, error) {
	rl := new(RateLimits)
	rl.members = make(map[string]*entry)

	if hz <= 0 {
		return nil, errors.New("hz must be greater than zero")
	}
	if accuracy <= 0 || accuracy > RL_MAX_BITS {
		return nil, fmt.Errorf("accuracy must be [1, %d]", RL_MAX_BITS)
	}
	if ffs(hz) < accuracy {
		accuracy = ffs(hz)
	}

	rl.hz = hz
	rl.bits = accuracy
	rl.size = 1 << accuracy
	rl.mask = rl.size - 1
	rl.offset = time.Duration(rl.size) * time.Second / time.Duration(hz)

	return rl, nil
}

func (rl *RateLimits) add(key string) (e *entry) {
	var (
		ok bool
	)

	rl.Lock()
	defer rl.Unlock()

	if e, ok = rl.members[key]; !ok {
		e = &entry{
			ts: make([]time.Time, rl.size),
			i:  0,
		}
		rl.members[key] = e
	}

	return e
}

func (rl *RateLimits) Update(key string) bool {
	rl.RLock()
	e, ok := rl.members[key]
	rl.RUnlock()

	if !ok {
		e = rl.add(key)
	}

	e.Lock()
	defer e.Unlock()
	now := time.Now()
	if now.Sub(e.ts[e.i&rl.mask]) <= rl.offset {
		return false
	}

	e.ts[e.i&rl.mask] = now
	e.i++
	return true
}
