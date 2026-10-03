//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package app

import "syscall"

// Replacing a checked regular file with a FIFO must not block OpenFile.
// The service checks the opened descriptor again before reading.
const readNoBlockFlag = syscall.O_NONBLOCK
