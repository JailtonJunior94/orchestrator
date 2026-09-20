package durable

import "github.com/JailtonJunior94/ai-spec-harness/internal/procref"

type ProcessRef = procref.ProcessRef

type LivenessResult = procref.LivenessResult

type LivenessProbe = procref.LivenessProbe

var CurrentProcessRef = procref.CurrentProcessRef

var DefaultLivenessProbe = procref.DefaultLivenessProbe
