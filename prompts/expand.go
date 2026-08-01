package prompts

import (
	"errors"
	"fmt"
	"text/template/parse"
)

var (
	// ErrCyclicTemplate は、partial が循環参照している場合に返されます。
	ErrCyclicTemplate = errors.New("テンプレートが循環参照しています")

	// ErrNotExpandable は、引数付きの {{template "x" .Foo}} のように
	// 展開するとデータコンテキストが変わってしまう参照があった場合に返されます。
	ErrNotExpandable = errors.New("この参照は展開できません")
)

// Expand は、指定されたモードの本文を、参照している partial を再帰的に
// 差し込んだソース文字列として返します。
//
// Build と違い {{.Field}} などのアクションは評価されずそのまま残ります。
// データを用意しなくても「実際に送られるプロンプトの構造」を確認できるため、
// プロンプトのカタログ表示や、本文に書かれた制約の検査に使えます。
// 同じ構文木から組み立てるので、結果は Build が使う本文と構造的に一致します。
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

// expandTemplate は、名前で引いたテンプレートの中身を展開済みのノード列として返します。
// visiting は現在たどっている経路で、循環参照の検出に使います。
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

// expandList は、ノード列を走査して {{template}} 参照を展開した新しいノード列を返します。
// 元の構文木は Build が使い続けるため、書き換えず新しい列を組み立てます。
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

// expandNode は、1つのノードを展開後のノード列へ変換します。
// {{if}} や {{range}} の内側にある参照も展開するため、分岐ノードは
// 値コピーしたうえで中身だけを差し替えます。
func (b *Builder) expandNode(node parse.Node, visiting map[string]bool) ([]parse.Node, error) {
	switch n := node.(type) {
	case *parse.TemplateNode:
		if !isDotPipe(n.Pipe) {
			return nil, fmt.Errorf("%w: '%s' に引数が指定されています", ErrNotExpandable, n.Name)
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

// expandBranch は、分岐ノードの本体と else 節をそれぞれ展開します。
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

// isDotPipe は、{{template "x"}} または {{template "x" .}} のように
// データコンテキストを変えない参照かどうかを判定します。
// {{template "x" .Foo}} を展開すると中身の .Bar が別のものを指してしまうため、
// そのような参照は展開せずエラーにします。
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
