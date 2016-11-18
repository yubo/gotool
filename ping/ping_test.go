/*
 * Copyright 2016 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package ping

import (
	"fmt"
	"testing"
)

func TestPing(t *testing.T) {
	if err := Run(1000); err != nil {
		t.Error(err)
	}

	ips := [][4]byte{
		{127, 0, 0, 1},
		{8, 8, 8, 8},
	}

	task := <-Go(ips, 1, 1, make(chan *Task, 1)).Done
	fmt.Printf("ret %v, err %v\n", task.Ret, task.Error)

	Kill()
}
