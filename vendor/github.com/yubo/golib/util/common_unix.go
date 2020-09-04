// +build darwin linux

package util

import "syscall"

func Kill(pid int, sig syscall.Signal) (err error) {
	return syscall.Kill(pid, sig)
}
