//go:build windows

package durable

func checkProcessLiveness(ref ProcessRef) LivenessResult {
	return LivenessResult{Reliable: false, Alive: false}
}
