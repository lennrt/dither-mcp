//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package app

// Platforms without POSIX FIFOs still validate the opened descriptor and use
// os.Root for path resolution. This flag deliberately adds no Unix dependency.
const readNoBlockFlag = 0
