// +build !linux,!darwin

package common

import "syscall"

func Kill(pid int, sig syscall.Signal) (err error) {
	return ErrUnsupported
}
