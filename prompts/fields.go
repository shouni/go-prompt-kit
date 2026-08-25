package prompts

import (
	"maps"
	"slices"
	"strings"
	"text/template/parse"
)

// Fields は、モードの本文が data に対して参照するフィールドを名前順で返します。
// 参照する partial の中身も、Expand と同じ規則で辿ります。
//
// missingkey=error のおかげで、data に無いフィールドは Build で必ずエラーになります。
// Fields はその手前で「このモードが何を要求しているか」をデータ無しで確かめるための
// 対になる関数です。
//
//	fields, err := builder.Fields("summarize")  // ["Language", "Source.Title"]
//
// ネストは "." で繋いだ形（"User.Name"）になります。
//
// range と with の本体は列挙しません。その内側では "." が別の値を指すため、
// フィールド名を data からの位置として報告できないからです（{{range .Items}} の
// .Items 自体、else 節、$ を起点にした {{$.Language}} は位置が確定するので含まれます）。
// 列挙されたものは確かに要求されていますが、列挙されなかったものが不要とは限りません。
//
// Expand と同じく、データコンテキストを変える {{template "x" .Foo}} は
// ErrNotExpandable、循環参照は ErrCyclicTemplate になります。
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

// collectFields は、ノード配下のフィールド参照を found へ集めます。
// rebound は "." が data 以外へ差し替わっている区間かどうかで、
// その区間では位置の確定する $ 起点の参照だけを拾います。
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
		// if は "." を変えないため、本体も else 節もそのまま辿ります。
		collectPipe(n.Pipe, rebound, found)
		collectFields(n.List, rebound, found)
		collectFields(n.ElseList, rebound, found)

	case *parse.RangeNode:
		collectBranch(n.BranchNode, rebound, found)

	case *parse.WithNode:
		collectBranch(n.BranchNode, rebound, found)
	}
}

// collectBranch は、range / with の各部を "." の差し替わりを踏まえて辿ります。
// 対象の式（{{range .Items}} の .Items）は外側の "." で評価されます。
// 本体の中では "." が要素・対象そのものへ差し替わり、
// else 節は対象が空だったときに実行されるため "." は変わりません。
func collectBranch(branch parse.BranchNode, rebound bool, found map[string]struct{}) {
	collectPipe(branch.Pipe, rebound, found)
	collectFields(branch.List, true, found)
	collectFields(branch.ElseList, rebound, found)
}

// collectPipe は、パイプラインの各引数からフィールド参照を集めます。
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

// collectArg は、パイプラインの引数1つを見てフィールド参照を集めます。
func collectArg(arg parse.Node, rebound bool, found map[string]struct{}) {
	switch a := arg.(type) {
	case *parse.FieldNode:
		// {{.User.Name}} — "." を起点とする参照です。
		if !rebound {
			addField(a.Ident, found)
		}

	case *parse.VariableNode:
		// {{$.Language}} — $ は常に data そのものを指すため、
		// range や with の内側でも位置が確定します。
		if len(a.Ident) > 1 && a.Ident[0] == "$" {
			addField(a.Ident[1:], found)
		}

	case *parse.ChainNode:
		// {{(.User).Name}} のように括弧を挟んだ形です。
		// 括弧の中身の位置が決まるなら繋いだ名前に、決まらないなら中身だけを見ます。
		if base, ok := chainBase(a.Node, rebound); ok {
			addField(concat(base, a.Field), found)
			return
		}
		collectArg(a.Node, rebound, found)

	case *parse.PipeNode:
		// 括弧で囲んだ入れ子のパイプラインです。
		collectPipe(a, rebound, found)
	}
}

// chainBase は、括弧の中身が data からの位置の決まる参照であれば、
// その位置を表す識別子の並びを返します。
func chainBase(node parse.Node, rebound bool) ([]string, bool) {
	switch n := node.(type) {
	case *parse.DotNode:
		// {{(.).Name}} — 差し替わっていなければ data そのものが基点です。
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
		// 括弧の中が単一の参照だけなら、その位置がそのまま基点になります。
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

// concat は、2つの識別子の並びを繋いだ新しい並びを返します。
// 構文木が持つスライスへ追記してしまわないよう、必ず新しい領域へ写します。
func concat(base, field []string) []string {
	ident := make([]string, 0, len(base)+len(field))
	ident = append(ident, base...)
	ident = append(ident, field...)

	return ident
}

// addField は、識別子の並びを "." で繋いだ名前として記録します。
func addField(ident []string, found map[string]struct{}) {
	if len(ident) == 0 {
		return
	}

	found[strings.Join(ident, ".")] = struct{}{}
}
