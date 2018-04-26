/*
 * Copyright 2016 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */
package flags

import "strings"

type Strings []string

func (i *Strings) String() string {
	return strings.Join([]string(*i), ",")
}

func (i *Strings) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Strings) Get() []string {
	return []string(*i)
}
