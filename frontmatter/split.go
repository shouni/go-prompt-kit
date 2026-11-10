// Package frontmatter splits the front matter at the top of a prompt file off its body.
//
// Front matter is a block of metadata between "---" lines, used to let a prompt carry its own
// information, such as a mode description. If it leaked into the body, the metadata would end up at
// the top of the instruction sent to the AI, so it is split off before the body is registered as a
// template.
//
//	files, _ := resource.Load(promptFiles, "prompts")
//	bodies, fronts := frontmatter.SplitMap(files)
//	infos, _ := frontmatter.DecodeMap[ModeInfo](fronts, yaml.Unmarshal)
//	builder, _ := prompts.NewBuilder(bodies)
//
// When the goal is loading and building templates, the same flow is also available by passing
// prompts.WithFrontMatter to prompts.LoadFS.
//
// This package does not interpret the metadata format. The caller passes the parsing function as an
// UnmarshalFunc (see UnmarshalFunc for why). The package itself depends on the standard library
// only.
package frontmatter

import "strings"

// delimiter delimits a front matter block. Both the opening and the closing delimiter must be a
// line consisting of this string alone.
const delimiter = "---"

// Split splits a leading "---\nmetadata\n---\n" block off the body. Without front matter, front is
// empty and body is the whole content.
//
// The closing delimiter is a line consisting of "---" alone. A line of a different length, such as
// "----", is not a delimiter. Only the delimiter line and the newline ending it are removed, so a
// blank line right after the delimiter stays as part of the body.
//
// The returned strings have their line endings normalized to LF and a leading BOM removed, whether
// or not front matter was found. A BOM or CRLF is invisible in an editor yet defeats front matter
// detection, so both are normalized before detecting.
func Split(content string) (front, body string) {
	normalized := normalize(content)

	opening := delimiter + "\n"
	if !strings.HasPrefix(normalized, opening) {
		return "", normalized
	}

	rest := normalized[len(opening):]

	// An empty front matter block: the closing line follows the opening line directly.
	if rest == delimiter {
		return "", ""
	}
	if strings.HasPrefix(rest, opening) {
		return "", rest[len(opening):]
	}

	closing := "\n" + delimiter
	if idx := strings.Index(rest, closing+"\n"); idx >= 0 {
		return rest[:idx], rest[idx+len(closing)+1:]
	}
	// In a file without a body, the closing delimiter is at the end of the file.
	if strings.HasSuffix(rest, closing) {
		return rest[:len(rest)-len(closing)], ""
	}

	// No closing delimiter. Even if it was meant as front matter, treat it as body.
	return "", normalized
}

// SplitMap runs Split on every entry of the map resource.Load returns and returns the bodies and
// the front matter in separate maps. The keys are the same as the input's; an entry without front
// matter has an empty front.
func SplitMap(files map[string]string) (bodies, fronts map[string]string) {
	bodies = make(map[string]string, len(files))
	fronts = make(map[string]string, len(files))

	for name, content := range files {
		front, body := Split(content)
		bodies[name] = body
		fronts[name] = front
	}

	return bodies, fronts
}

// bom is the UTF-8 byte order mark.
const bom = "\ufeff"

// normalize removes invisible differences (BOM, CRLF) that would defeat detection.
func normalize(content string) string {
	return strings.ReplaceAll(strings.TrimPrefix(content, bom), "\r\n", "\n")
}
