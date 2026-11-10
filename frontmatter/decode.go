package frontmatter

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ErrNoUnmarshalFunc is returned when no parsing function is given.
var ErrNoUnmarshalFunc = errors.New("frontmatter: no unmarshal function configured")

// UnmarshalFunc reads front matter into a struct. yaml.Unmarshal and json.Unmarshal can be passed
// as is.
//
// This package takes the YAML library from the caller instead of referencing one directly so that
// consumers can choose it. A fixed library would force every switch to wait for a release of this
// module, and mid-migration a single binary would carry two implementations.
type UnmarshalFunc func(data []byte, v any) error

// Decode reads the front matter returned by Split into v. Empty front matter is a no-op returning
// nil, so prompts without metadata are handled as they are.
func Decode(front string, v any, unmarshal UnmarshalFunc) error {
	// Check this first regardless of the data, so that a wiring mistake does not go unnoticed when
	// only empty entries are passed.
	if unmarshal == nil {
		return ErrNoUnmarshalFunc
	}
	if strings.TrimSpace(front) == "" {
		return nil
	}

	if err := unmarshal([]byte(front), v); err != nil {
		return fmt.Errorf("parse front matter: %w", err)
	}

	return nil
}

// DecodeAs reads the front matter returned by Split as a T and returns it. It works like Decode,
// except that no variable has to be prepared first (it is the single-value form of DecodeMap).
//
//	info, err := frontmatter.DecodeAs[ModeInfo](front, yaml.Unmarshal)
//
// Empty front matter returns T's zero value, not an error.
func DecodeAs[T any](front string, unmarshal UnmarshalFunc) (T, error) {
	var v T
	if err := Decode(front, &v, unmarshal); err != nil {
		var zero T
		return zero, err
	}

	return v, nil
}

// DecodeMap reads all the front matter returned by SplitMap into T. The keys are the same as the
// input's; an entry without front matter gets T's zero value.
//
// If an entry fails to parse, the error names its key (so that malformed metadata cannot pass
// through as blank). To treat entries differently, use Split and Decode directly.
func DecodeMap[T any](fronts map[string]string, unmarshal UnmarshalFunc) (map[string]T, error) {
	if unmarshal == nil {
		return nil, ErrNoUnmarshalFunc
	}

	out := make(map[string]T, len(fronts))
	// Process in name order so that the reported key does not change when several entries are
	// broken.
	for _, name := range slices.Sorted(maps.Keys(fronts)) {
		var v T
		if err := Decode(fronts[name], &v, unmarshal); err != nil {
			return nil, fmt.Errorf("'%s': %w", name, err)
		}
		out[name] = v
	}

	return out, nil
}
