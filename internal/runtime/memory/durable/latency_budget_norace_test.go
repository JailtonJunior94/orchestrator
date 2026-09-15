//go:build !race

package durable_test

import "time"

const buildContextP95Budget = 200 * time.Millisecond
