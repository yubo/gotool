/*
 * Copyright 2016 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */
package list

import (
	"fmt"
	"testing"
	"unsafe"
)

type foo struct {
	x    int
	list ListHead
}

func (list *ListHead) entry() *foo {
	return (*foo)(unsafe.Pointer((uintptr(unsafe.Pointer(list)) -
		unsafe.Offsetof(((*foo)(nil)).list))))
}

func list2foo(list *ListHead) *foo {
	return (*foo)(unsafe.Pointer((uintptr(unsafe.Pointer(list)) -
		unsafe.Offsetof(((*foo)(nil)).list))))
}

func TestAdd(t *testing.T) {
	h := &ListHead{}

	h.Init()

	for i := 0; i < 10; i++ {
		fo := foo{x: i}
		h.Add(&fo.list)
	}

	for p := h.Next; p != h; p = p.Next {
		fmt.Printf("x:%d\n", p.entry().x)
	}
}
