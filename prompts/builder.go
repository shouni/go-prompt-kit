// Package prompts provides a builder that builds the prompt for a mode from embedded templates.
package prompts

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"text/template"
)

// DefaultPartialPrefix is the default prefix marking a partial template. Entries with this prefix
// are not exposed as modes; they are registered only to be referenced from other templates with
// {{template "_name" .}}. WithPartialPrefix changes it.
const DefaultPartialPrefix = "_"

var (
	// ErrEmptyTemplates is returned when NewBuilder is given an empty or nil map.
	ErrEmptyTemplates = errors.New("prompts: template map is empty or nil")

	// ErrUnknownMode is returned when Build is called with a mode that is not registered.
	ErrUnknownMode = errors.New("prompts: unknown mode")

	// ErrDuplicateDefinition is returned when several entries {{define}} a template with the same
	// name. All templates share one namespace, so left alone the last one would silently overwrite
	// the others.
	ErrDuplicateDefinition = errors.New("prompts: duplicate template definition")

	// ErrLoadOnlyOption is returned when an option that only means something while loading is
	// passed to NewBuilder. Ignoring it silently would hide that the intended filtering is not in
	// effect.
	ErrLoadOnlyOption = errors.New("prompts: option is only valid with LoadFS")
)

// Builder holds a loaded set of templates and the rules for choosing a mode. Every template is
// registered as associated with one namespace, so a mode body can reference a partial with
// {{template "_name" .}}.
//
// A Builder is immutable after construction, and every exported method can be called from several
// goroutines at once. Where it is called concurrently, such as from HTTP handlers, build it once at
// startup and reuse it.
type Builder struct {
	root        *template.Template
	modes       []string
	modeSet     map[string]struct{}
	defaultMode string
	// fronts holds the front matter split off by WithFrontMatter (entry name -> raw string).
	fronts map[string]string
}

// NewBuilder builds a Builder from a map of templates keyed by mode name. Entries starting with
// DefaultPartialPrefix are treated as partials (see IsPartial).
//
// Options about loading are for LoadFS only; passing one returns ErrLoadOnlyOption.
func NewBuilder(templates map[string]string, opts ...Option) (*Builder, error) {
	cfg := newConfig(opts...)

	if len(cfg.loadOnly) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrLoadOnlyOption, strings.Join(cfg.loadOnly, ", "))
	}

	return newBuilder(templates, cfg)
}

// newBuilder builds a Builder from applied settings. LoadFS consumes the load-only options itself,
// so it calls this directly, bypassing NewBuilder's check.
func newBuilder(templates map[string]string, cfg *config) (*Builder, error) {
	if len(templates) == 0 {
		return nil, ErrEmptyTemplates
	}

	// missingkey=error is set on root. Options are shared among associated templates, so this alone
	// applies to every entry, partials included.
	root := template.New("").Option("missingkey=error")
	if len(cfg.funcs) > 0 {
		root = root.Funcs(cfg.funcs)
	}

	modes := make([]string, 0, len(templates))
	modeSet := make(map[string]struct{}, len(templates))
	// Template name -> the entry that defined it (for detecting duplicate definitions; see
	// parseEntry).
	definedBy := make(map[string]string, len(templates))

	// Process in name order to keep the order of reported errors stable.
	for _, name := range slices.Sorted(maps.Keys(templates)) {
		content := templates[name]
		if content == "" {
			return nil, fmt.Errorf("load prompt template '%s': content is empty", name)
		}
		// Trim only after checking for emptiness. A newline-only partial is registered as "a
		// partial that outputs nothing" (which is different from empty content).
		if cfg.trimPartials && IsPartial(name, cfg.partialPrefix) {
			content = strings.TrimRight(content, "\r\n")
		}

		parsed, err := parseEntry(name, content, cfg.funcs, definedBy)
		if err != nil {
			return nil, err
		}
		if err := attach(root, parsed); err != nil {
			return nil, fmt.Errorf("register prompt '%s': %w", name, err)
		}

		if !IsPartial(name, cfg.partialPrefix) {
			modes = append(modes, name)
			modeSet[name] = struct{}{}
		}
	}

	if len(modes) == 0 {
		return nil, fmt.Errorf("%w: no templates other than partials", ErrEmptyTemplates)
	}

	// The default mode must be an existing mode (catches typos at construction).
	if cfg.defaultMode != "" {
		if _, ok := modeSet[cfg.defaultMode]; !ok {
			return nil, fmt.Errorf("default mode '%s' is not registered: %w", cfg.defaultMode, ErrUnknownMode)
		}
	}

	return &Builder{
		root:        root,
		modes:       modes,
		modeSet:     modeSet,
		defaultMode: cfg.defaultMode,
	}, nil
}

// Build executes the template of the given mode with data. When WithDefaultMode is set, an
// unregistered mode (including the empty string) falls back to the default mode.
//
// A field the template references that is missing from data is an error (it is never silently
// output as "<no value>"). The validity of data's contents is not checked; checking the values is
// the caller's responsibility.
func (b *Builder) Build(mode string, data any) (string, error) {
	resolved, err := b.resolveMode(mode)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := b.root.ExecuteTemplate(&sb, resolved, data); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}

	return sb.String(), nil
}

// resolveMode decides the name of the mode to run: the mode itself if registered, otherwise the
// default mode, and ErrUnknownMode when no default mode is set.
func (b *Builder) resolveMode(mode string) (string, error) {
	if _, ok := b.modeSet[mode]; ok {
		return mode, nil
	}
	if b.defaultMode != "" {
		return b.defaultMode, nil
	}
	return "", fmt.Errorf("%w: '%s'", ErrUnknownMode, mode)
}

// Modes returns the mode names Build accepts, in name order (partials excluded).
func (b *Builder) Modes() []string {
	return slices.Clone(b.modes)
}

// Has reports whether the given mode is registered. It ignores the fallback to the default mode
// (WithDefaultMode) and reports actual registration.
func (b *Builder) Has(mode string) bool {
	_, ok := b.modeSet[mode]
	return ok
}

// FrontMatter returns the front matter of the given entry as the raw string that was split off. It
// has a value only when WithFrontMatter was passed to LoadFS. An entry without front matter and an
// unknown entry both return the empty string.
//
// This package does not interpret the format; pass it to frontmatter.Decode / DecodeAs.
func (b *Builder) FrontMatter(name string) string {
	return b.fronts[name]
}

// FrontMatters returns the split-off front matter as a map keyed by entry name (partials included).
// It can be passed to frontmatter.DecodeMap as is. Changes the caller makes do not affect the
// Builder.
func (b *Builder) FrontMatters() map[string]string {
	return maps.Clone(b.fronts)
}

// parseEntry parses an entry on its own and records the template names it defines. It returns an
// error when another entry already defined the same name. All entries share one namespace, so
// without this check the last one would silently overwrite the others.
//
// The check uses the isolated parse because walking root cannot tell these names apart from names
// another entry already defined. A {{template "x"}} reference creates an entry without a parse
// tree, so only actual {{define "x"}} definitions and the entry's own name are handled here.
func parseEntry(entry, content string, funcs template.FuncMap, definedBy map[string]string) (*template.Template, error) {
	scratch := template.New(entry)
	if len(funcs) > 0 {
		scratch = scratch.Funcs(funcs)
	}
	if _, err := scratch.Parse(content); err != nil {
		return nil, fmt.Errorf("parse prompt '%s': %w", entry, err)
	}

	for _, tmpl := range scratch.Templates() {
		name := tmpl.Name()
		if name == "" || tmpl.Tree == nil {
			continue
		}

		if owner, seen := definedBy[name]; seen && owner != entry {
			return nil, fmt.Errorf("%w: '%s' is defined in both '%s' and '%s'",
				ErrDuplicateDefinition, name, owner, entry)
		}
		definedBy[name] = entry
	}

	return scratch, nil
}

// attach moves the parse trees from an isolated parse into the shared namespace. It only hands the
// trees over, so the same content is never parsed twice. Entries without a parse tree (references
// to undefined templates via {{template "x"}}) are skipped so that they do not erase a defined
// tree.
func attach(root *template.Template, parsed *template.Template) error {
	for _, tmpl := range parsed.Templates() {
		if tmpl.Tree == nil {
			continue
		}
		if _, err := root.AddParseTree(tmpl.Name(), tmpl.Tree); err != nil {
			return err
		}
	}
	return nil
}

// IsPartial reports whether a template name refers to a partial (a reference-only piece, not a
// mode). It tests the last path element, since recursive loading produces path-shaped names such as
// "en/_output". An empty prefix disables partial detection (every entry becomes a mode).
//
// Builder uses the same rule to decide what Modes() and Build() cover. When filtering a loaded map
// yourself before handing it to a Builder, use this function instead of writing the predicate
// twice.
//
//	if prompts.IsPartial(name, prompts.DefaultPartialPrefix) { continue }
func IsPartial(name, prefix string) bool {
	if prefix == "" {
		return false
	}
	return strings.HasPrefix(path.Base(name), prefix)
}
