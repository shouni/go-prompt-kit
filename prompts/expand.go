package prompts

import (
	"errors"
	"fmt"
	"text/template/parse"
)

var (
	// ErrCyclicTemplate is returned when partials reference each other in a cycle.
	ErrCyclicTemplate = errors.New("prompts: template reference is cyclic")

	// ErrNotExpandable is returned for a reference that would change the data context if expanded,
	// such as {{template "x" .Foo}} with an argument.
	ErrNotExpandable = errors.New("prompts: reference is not expandable")
)

// Expand returns the body of the given mode as source text with the partials it references inserted
// recursively.
//
// Unlike Build, actions such as {{.Field}} are not evaluated and stay as they are. It shows "the
// structure of the prompt actually sent" without preparing data, so it is useful for prompt
// catalogs and for checking constraints written in a body. It is assembled from the same parse
// trees, so the result matches the body Build uses structurally.
func (b *Builder) Expand(mode string) (string, error) {
	resolved, err := b.resolveMode(mode)
	if err != nil {
		return "", err
	}

	list, err := b.expandTemplate(resolved, map[string]bool{})
	if err != nil {
		return "", err
	}

	return list.String(), nil
}

// expandTemplate returns the contents of the named template as an expanded node list. visiting is
// the path currently being followed, used to detect cycles.
func (b *Builder) expandTemplate(name string, visiting map[string]bool) (*parse.ListNode, error) {
	tmpl := b.root.Lookup(name)
	if tmpl == nil || tmpl.Tree == nil {
		return nil, fmt.Errorf("%w: '%s'", ErrUnknownMode, name)
	}

	if visiting[name] {
		return nil, fmt.Errorf("%w: '%s'", ErrCyclicTemplate, name)
	}
	visiting[name] = true
	defer delete(visiting, name)

	return b.expandList(tmpl.Root, visiting)
}

// expandList walks a node list and returns a new node list with {{template}} references expanded.
// Build keeps using the original parse trees, so a new list is built instead of rewriting them.
func (b *Builder) expandList(list *parse.ListNode, visiting map[string]bool) (*parse.ListNode, error) {
	expanded := &parse.ListNode{NodeType: parse.NodeList}
	if list == nil {
		return expanded, nil
	}

	for _, node := range list.Nodes {
		nodes, err := b.expandNode(node, visiting)
		if err != nil {
			return nil, err
		}
		expanded.Nodes = append(expanded.Nodes, nodes...)
	}

	return expanded, nil
}

// expandNode turns one node into the expanded node list. References inside {{if}} and {{range}} are
// expanded too, so a branch node is copied by value and only its contents replaced.
func (b *Builder) expandNode(node parse.Node, visiting map[string]bool) ([]parse.Node, error) {
	switch n := node.(type) {
	case *parse.TemplateNode:
		if !isDotPipe(n.Pipe) {
			return nil, fmt.Errorf("%w: '%s' is given an argument", ErrNotExpandable, n.Name)
		}
		sub, err := b.expandTemplate(n.Name, visiting)
		if err != nil {
			return nil, err
		}
		return sub.Nodes, nil

	case *parse.IfNode:
		branch, err := b.expandBranch(n.BranchNode, visiting)
		if err != nil {
			return nil, err
		}
		clone := *n
		clone.BranchNode = branch
		return []parse.Node{&clone}, nil

	case *parse.RangeNode:
		branch, err := b.expandBranch(n.BranchNode, visiting)
		if err != nil {
			return nil, err
		}
		clone := *n
		clone.BranchNode = branch
		return []parse.Node{&clone}, nil

	case *parse.WithNode:
		branch, err := b.expandBranch(n.BranchNode, visiting)
		if err != nil {
			return nil, err
		}
		clone := *n
		clone.BranchNode = branch
		return []parse.Node{&clone}, nil

	case *parse.ListNode:
		sub, err := b.expandList(n, visiting)
		if err != nil {
			return nil, err
		}
		return sub.Nodes, nil

	default:
		return []parse.Node{node}, nil
	}
}

// expandBranch expands a branch node's body and its else clause.
func (b *Builder) expandBranch(branch parse.BranchNode, visiting map[string]bool) (parse.BranchNode, error) {
	body, err := b.expandList(branch.List, visiting)
	if err != nil {
		return branch, err
	}
	branch.List = body

	if branch.ElseList != nil {
		elseBody, err := b.expandList(branch.ElseList, visiting)
		if err != nil {
			return branch, err
		}
		branch.ElseList = elseBody
	}

	return branch, nil
}

// isDotPipe reports whether a reference keeps the data context, like {{template "x"}} or {{template
// "x" .}}. Expanding {{template "x" .Foo}} would make the .Bar inside point at something else, so
// such a reference is an error instead of being expanded.
func isDotPipe(pipe *parse.PipeNode) bool {
	if pipe == nil {
		return true
	}
	if len(pipe.Decl) > 0 || len(pipe.Cmds) != 1 {
		return false
	}

	cmd := pipe.Cmds[0]
	if len(cmd.Args) != 1 {
		return false
	}
	_, isDot := cmd.Args[0].(*parse.DotNode)

	return isDot
}
