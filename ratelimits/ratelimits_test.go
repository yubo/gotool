/*
 * Copyright 2017 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */
package ratelimits

import (
	"fmt"
	"testing"
	"time"
)

func test(qps, accuracy, ts uint32) {
	stats := make(map[bool]int)
	x, _ := New(qps, accuracy)
	done := make(chan struct{}, 1)
	go func(done chan struct{}) {
		for {
			select {
			case <-done:
				fmt.Printf("true:%fHz false:%fHz\n",
					float64(stats[true])/float64(ts),
					float64(stats[false])/float64(ts))
				return
			default:
				ok := x.Update("test")
				stats[ok]++
				time.Sleep(time.Nanosecond)
			}
		}
	}(done)
	time.Sleep(time.Second * time.Duration(ts))
	done <- struct{}{}
}

func TestAll(t *testing.T) {
	test(10, 1, 2)
	test(10, 2, 2)
	test(10, 4, 2)
	test(10, 8, 2)
	test(100, 1, 2)
	test(100, 2, 2)
	test(100, 4, 2)
	test(100, 8, 2)
	test(10000, 1, 1)
	test(10000, 2, 1)
	test(10000, 4, 1)
	test(10000, 8, 1)
}
