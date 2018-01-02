package main

import (
	"bytes"
	"fmt"
	"os"
)

func Err(exitStatus int, err error, format string, args ...interface{}) {
	b := bytes.Buffer{}

	b.WriteString("error: ")
	b.WriteString(fmt.Sprintf(format, args...))

	if err != nil {
		b.WriteString(fmt.Sprintf(": %v", err))
	}

	os.Stdout.Sync()
	fmt.Fprintf(os.Stderr, "%s\n", b.String())
	os.Stderr.Sync()

	if exitStatus != 0 {
		os.Exit(exitStatus)
	}
}
