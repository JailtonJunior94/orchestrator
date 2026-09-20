package hooks

import "github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"

var canonicalEventByPoint = map[string]hookcontract.EventKind{
	PointRuntimePreOpen:       hookcontract.EventSessionStart,
	PointToolCallPreDispatch:  hookcontract.EventBeforeTool,
	PointToolCallPostComplete: hookcontract.EventAfterTool,
	PointSessionPostEnd:       hookcontract.EventBeforeComplete,
}

var pointsWithoutCanonicalEvent = []string{
	PointPromptPreBuild,
	PointPromptPostBuild,
	PointSessionPostReview,
	PointMemoryFactRecorded,
	PointMemoryFactArchived,
	PointMemoryFactPromoted,
	PointMemoryContradictionDetected,
	PointMemorySecretRedacted,
	PointMemoryCompactionExecuted,
	PointMemoryBatonTransferred,
}

func CanonicalEventFor(point string) (hookcontract.EventKind, bool) {
	event, ok := canonicalEventByPoint[point]
	return event, ok
}

func PointsWithoutCanonicalEvent() []string {
	out := make([]string, len(pointsWithoutCanonicalEvent))
	copy(out, pointsWithoutCanonicalEvent)
	return out
}

func AllPoints() []string {
	out := make([]string, 0, len(canonicalEventByPoint)+len(pointsWithoutCanonicalEvent))
	for point := range canonicalEventByPoint {
		out = append(out, point)
	}
	out = append(out, pointsWithoutCanonicalEvent...)
	return out
}
