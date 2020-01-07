package events

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

func isInternal(pkg string) bool {
	pkgs := []string{"runtime", "logger", "events"}

	for _, p := range pkgs {
		if strings.Contains(pkg, p) {
			return true

		}
	}

	return false
}

func getCaller(exclude func(string) bool) string {
	toPC := 10
	pc := make([]uintptr, toPC)
	n := runtime.Callers(0, pc)
	if n == 0 {
		return "unknown.caller"
	}

	frames := runtime.CallersFrames(pc[0:toPC])
	var frame runtime.Frame
	callerName := ""
	for {
		var more bool
		frame, more = frames.Next()
		if !more {
			break
		}
		if len(frame.Function) == 0 {
			break
		}
		fParts := strings.Split(frame.Function, ".")
		pkgName := strings.Join(fParts[:len(fParts)-1], ".")
		if !exclude(pkgName) {
			callerName = frame.Function
			break
		}

	}

	if len(callerName) == 0 {
		callerName = "unknown.func"
	}
	return fmt.Sprintf("%s@%s:%d ", callerName, filepath.Base(frame.File), frame.Line)
}
