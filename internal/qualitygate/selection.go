package qualitygate

import "fmt"

type Selection struct {
	required []CheckKind
	optional []CheckKind
}

func NewSelection(required, optional []CheckKind) (Selection, error) {
	seen := make(map[CheckKind]string, len(required)+len(optional))
	for _, kind := range required {
		if !kind.Valid() {
			return Selection{}, fmt.Errorf("%w: %d", ErrUnknownCheckKind, int(kind))
		}
		if origin, exists := seen[kind]; exists {
			return Selection{}, fmt.Errorf("qualitygate: check %s declared in both %s and required", kind, origin)
		}
		seen[kind] = "required"
	}
	for _, kind := range optional {
		if !kind.Valid() {
			return Selection{}, fmt.Errorf("%w: %d", ErrUnknownCheckKind, int(kind))
		}
		if origin, exists := seen[kind]; exists {
			return Selection{}, fmt.Errorf("qualitygate: check %s declared in both %s and optional", kind, origin)
		}
		seen[kind] = "optional"
	}

	return Selection{
		required: append([]CheckKind(nil), required...),
		optional: append([]CheckKind(nil), optional...),
	}, nil
}

func (s Selection) Required() []CheckKind {
	return append([]CheckKind(nil), s.required...)
}

func (s Selection) Optional() []CheckKind {
	return append([]CheckKind(nil), s.optional...)
}
