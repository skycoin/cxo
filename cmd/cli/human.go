package main

import (
	"fmt"
	"strings"
	"time"
)

func humanDuration(d time.Duration, zero string) string {
	if d == 0 {
		return zero
	}
	return fmt.Sprint(d)
}

// yes/no
func humanBool(t bool) string {
	if t {
		return "yes"
	}
	return "no"
}

// humanMemory returns human readable memory string
func humanMemory(bytes int) string {
	var fb float64 = float64(bytes)
	var ms string = "B"
	for _, m := range []string{"KiB", "MiB", "GiB"} {
		if fb > 1024.0 {
			fb = fb / 1024.0
			ms = m
			continue
		}
		break
	}
	if ms == "B" {
		return fmt.Sprintf("%.0fB", fb)
	}
	// 2.00 => 2
	// 2.10 => 2.1
	// 2.53 => 2.53
	return strings.TrimRight(
		strings.TrimRight(fmt.Sprintf("%.2f", fb), "0"),
		".") + ms
}
