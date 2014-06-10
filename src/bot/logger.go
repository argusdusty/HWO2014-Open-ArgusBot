package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

func logMessage(debug int, v ...interface{}) {
	if debug <= DEBUGMODE {
		fmt.Print(time.Now().Format(time.StampNano + " "))
		fmt.Println(v...)
	}
}

func logError(err error) {
	logMessage(0, "Error:", err)
}

func logStackTrace() {
	out := string(debug.Stack())
	for _, line := range strings.Split(out, "\n") {
		logMessage(0, "Stack trace:", line)
	}
}

func logFatal(err error) {
	logError(err)
	logStackTrace()
	os.Exit(1)
}
