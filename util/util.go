package util

import (
	"fmt"
	"os"
)

func Warningf(format string, a ...interface{}) {
	format = "WARNING: " + format + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}
