//go:build race

package durable_test

import "time"

const buildContextP95Budget = 400 * time.Millisecond
