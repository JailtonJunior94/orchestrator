package durable

import (
	"os"
	"time"
)

type ProcessRef struct {
	PID       int
	Hostname  string
	StartedAt time.Time
}

func CurrentProcessRef() ProcessRef {
	hostname, _ := os.Hostname()
	return ProcessRef{
		PID:       os.Getpid(),
		Hostname:  hostname,
		StartedAt: time.Now(),
	}
}

type LivenessResult struct {
	Reliable bool
	Alive    bool
}

type LivenessProbe interface {
	Probe(ref ProcessRef) LivenessResult
}

type processLivenessProbe struct{}

func (processLivenessProbe) Probe(ref ProcessRef) LivenessResult {
	return checkProcessLiveness(ref)
}

var DefaultLivenessProbe LivenessProbe = processLivenessProbe{}
