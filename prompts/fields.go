package prompts

import (
	"maps"
	"slices"
	"strings"
	"text/template/parse"
)

// Fields returns, in name order, the fields the mode's body references in data. It follows the
// contents of referenced partials by the same rules as Expand.
//
// Thanks to missingkey=error, a field missing from data always fails Build. Fields is its
// counterpart for checking "what this mode requires" ahead of that, without data.
//
//	fields, err := builder.Fields("summarize")  // ["Language", "Source.Title"]
//
// Nested fields are joined with "." ("User.Name").
//
// The bodies of range and with are not listed: inside them "." points at another value, so a field
// name there cannot be reported as a position in data. (The .Items of {{range .Items}} itself, the
// else clause, and $-rooted references such as {{$.Language}} have a fixed position and are
// included.) What is listed is certainly required, but what is not listed is not necessarily
// unneeded.
//
// As with Expand, a {{template "x" .Foo}} that changes the data context gives ErrNotExpandable, and
// a cycle gives ErrCyclicTemplate.
func (b *Builder) Fields(mode string) ([]string, error) {
	resolved, err := b.resolveMode(mode)
	if err != nil {
		return nil, err
	}

	list, err := b.expandTemplate(resolved, map[string]bool{})
	if err != nil {
		return nil, err
	}

	found := make(map[string]struct{})
	collectFields(list, false, found)

	return slices.Sorted(maps.Keys(found)), nil
}

// collectFields collects the field references under node into found. rebound says whether "." has
// been rebound to something other than data; in such a span only $-rooted references, whose
// position is fixed, are picked up.
func collectFields(node parse.Node, rebound bool, found map[string]struct{}) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			collectFields(child, rebound, found)
		}

	case *parse.ActionNode:
		collectPipe(n.Pipe, rebound, found)

	case *parse.IfNode:
		// if does not change ".", so both the body and the else clause are followed as they are.
		collectPipe(n.Pipe, rebound, found)
		collectFields(n.List, rebound, found)
		collectFields(n.ElseList, rebound, found)

	case *parse.RangeNode:
		collectBranch(n.BranchNode, rebound, found)

	case *parse.WithNode:
		collectBranch(n.BranchNode, rebound, found)
	}
}

// collectBranch follows the parts of a range / with, taking the rebinding of "." into account. The
// target expression (the .Items of {{range .Items}}) is evaluated with the outer ".". Inside the
// body "." is rebound to the element or the target itself, while the else clause runs when the
// target is empty, so "." does not change there.
func collectBranch(branch parse.BranchNode, rebound bool, found map[string]struct{}) {
	collectPipe(branch.Pipe, rebound, found)
	collectFields(branch.List, true, found)
	collectFields(branch.ElseList, rebound, found)
}

// collectPipe collects field references from every argument of a pipeline.
func collectPipe(pipe *parse.PipeNode, rebound bool, found map[string]struct{}) {
	if pipe == nil {
		return
	}

	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			collectArg(arg, rebound, found)
		}
	}
}

// collectArg collects field references from one pipeline argument.
func collectArg(arg parse.Node, rebound bool, found map[string]struct{}) {
	switch a := arg.(type) {
	case *parse.FieldNode:
		// {{.User.Name}} — a reference rooted at ".".
		if !rebound {
			addField(a.Ident, found)
		}

	case *parse.VariableNode:
		// {{$.Language}} — $ always points at data itself, so the position is fixed even inside
		// range or with.
		if len(a.Ident) > 1 && a.Ident[0] == "$" {
			addField(a.Ident[1:], found)
		}

	case *parse.ChainNode:
		// A parenthesized form such as {{(.User).Name}}. If the position of the parenthesized part
		// is fixed, record the joined name; otherwise look only at the inner part.
		if base, ok := chainBase(a.Node, rebound); ok {
			addField(concat(base, a.Field), found)
			return
		}
		collectArg(a.Node, rebound, found)

	case *parse.PipeNode:
		// A nested, parenthesized pipeline.
		collectPipe(a, rebound, found)
	}
}

// chainBase returns the identifiers for the position when the parenthesized part is a reference
// with a fixed position in data.
func chainBase(node parse.Node, rebound bool) ([]string, bool) {
	switch n := node.(type) {
	case *parse.DotNode:
		// {{(.).Name}} — unless rebound, data itself is the base.
		return nil, !rebound

	case *parse.FieldNode:
		if rebound {
			return nil, false
		}
		return n.Ident, true

	case *parse.VariableNode:
		if len(n.Ident) == 0 || n.Ident[0] != "$" {
			return nil, false
		}
		return n.Ident[1:], true

	case *parse.PipeNode:
		// If the parentheses hold a single reference, its position is the base as is.
		if len(n.Decl) > 0 || len(n.Cmds) != 1 {
			return nil, false
		}
		cmd := n.Cmds[0]
		if len(cmd.Args) != 1 {
			return nil, false
		}
		return chainBase(cmd.Args[0], rebound)
	}

	return nil, false
}

// concat returns a new sequence joining two identifier sequences. It always copies into new storage
// so that it never appends to a slice owned by the parse tree.
func concat(base, field []string) []string {
	ident := make([]string, 0, len(base)+len(field))
	ident = append(ident, base...)
	ident = append(ident, field...)

	return ident
}

// addField records an identifier sequence as a name joined with ".".
func addField(ident []string, found map[string]struct{}) {
	if len(ident) == 0 {
		return
	}

	found[strings.Join(ident, ".")] = struct{}{}
}
