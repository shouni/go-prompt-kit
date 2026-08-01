// Package prompts は、埋め込みテンプレートからモードに応じたプロンプトを
// 構築するビルダーを提供します。
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

// DefaultPartialPrefix は、部品テンプレート（partial）を示す既定の接頭辞です。
// この接頭辞を持つエントリはモードとして公開されず、
// 他のテンプレートから {{template "_name" .}} で参照するためだけに登録されます。
// WithPartialPrefix で変更できます。
const DefaultPartialPrefix = "_"

var (
	// ErrEmptyTemplates は、NewBuilder に空またはnilのマップが渡された場合に返されます。
	ErrEmptyTemplates = errors.New("テンプレートマップが空またはnilです")

	// ErrUnknownMode は、登録されていないモードで Build が呼ばれた場合に返されます。
	ErrUnknownMode = errors.New("不明なモードです")

	// ErrDuplicateDefinition は、複数のエントリが同じ名前のテンプレートを
	// {{define}} で定義している場合に返されます。
	// 全テンプレートは1つの名前空間を共有するため、放置すると後勝ちで静かに上書きされます。
	ErrDuplicateDefinition = errors.New("テンプレート定義が重複しています")
)

// Builder はプロンプトの構成を管理し、モード選択のロジックを内包します。
// すべてのテンプレートは1つの名前空間へ関連付けて登録されるため、
// モード本文から partial を {{template "_name" .}} で参照できます。
type Builder struct {
	root        *template.Template
	modes       []string
	modeSet     map[string]struct{}
	defaultMode string
}

// NewBuilder は Builder を初期化します。
// キー（パス形式の場合は末尾の要素）が接頭辞 "_" で始まるエントリは partial として扱われ、
// Build の対象にはならず、他のテンプレートからの参照専用になります
// （接頭辞は WithPartialPrefix で変更できます）。
func NewBuilder(templates map[string]string, opts ...Option) (*Builder, error) {
	if len(templates) == 0 {
		return nil, ErrEmptyTemplates
	}

	cfg := newConfig(opts...)

	// Option("missingkey=error") を追加して、変数の埋め込み漏れを許容しない。
	// option は関連テンプレート間で共有されるため、root への設定が全体へ効きます。
	root := template.New("").Option("missingkey=error")
	if len(cfg.funcs) > 0 {
		root = root.Funcs(cfg.funcs)
	}

	modes := make([]string, 0, len(templates))
	modeSet := make(map[string]struct{}, len(templates))
	// テンプレート名 -> それを定義したエントリ名。
	// 全エントリが1つの名前空間を共有するため、重複定義を検出して静かな上書きを防ぎます。
	definedBy := make(map[string]string, len(templates))

	// エラー報告の順序を安定させるため、名前順に処理します。
	for _, name := range slices.Sorted(maps.Keys(templates)) {
		content := templates[name]
		if content == "" {
			return nil, fmt.Errorf("プロンプトテンプレート '%s' の読み込みに失敗しました: 内容が空です", name)
		}

		if err := recordDefinitions(name, content, cfg.funcs, definedBy); err != nil {
			return nil, err
		}

		if _, err := root.New(name).Parse(content); err != nil {
			return nil, fmt.Errorf("プロンプト '%s' の解析に失敗: %w", name, err)
		}

		if !isPartial(name, cfg.partialPrefix) {
			modes = append(modes, name)
			modeSet[name] = struct{}{}
		}
	}

	if len(modes) == 0 {
		return nil, fmt.Errorf("%w: partial以外のテンプレートがありません", ErrEmptyTemplates)
	}

	// 既定モードは実在するモードでなければなりません（typoを構築時に検出します）。
	if cfg.defaultMode != "" {
		if _, ok := modeSet[cfg.defaultMode]; !ok {
			return nil, fmt.Errorf("既定モード '%s' が登録されていません: %w", cfg.defaultMode, ErrUnknownMode)
		}
	}

	return &Builder{
		root:        root,
		modes:       modes,
		modeSet:     modeSet,
		defaultMode: cfg.defaultMode,
	}, nil
}

// Build は、要求されたモードに応じて適切なテンプレートを実行します。
// WithDefaultMode が指定されている場合、未登録のモード（空文字を含む）は既定モードへ委ねられます。
// 注意: data の内容に関する事前バリデーションは行いません。呼び出し元で適切なデータが設定されていることを保証してください。
func (b *Builder) Build(mode string, data any) (string, error) {
	resolved, err := b.resolveMode(mode)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := b.root.ExecuteTemplate(&sb, resolved, data); err != nil {
		return "", fmt.Errorf("プロンプトテンプレートの実行に失敗しました: %w", err)
	}

	return sb.String(), nil
}

// resolveMode は、実行するモード名を決定します。
// 登録済みならそのまま、未登録なら既定モード、既定モードが未設定なら ErrUnknownMode を返します。
func (b *Builder) resolveMode(mode string) (string, error) {
	if _, ok := b.modeSet[mode]; ok {
		return mode, nil
	}
	if b.defaultMode != "" {
		return b.defaultMode, nil
	}
	return "", fmt.Errorf("%w: '%s'", ErrUnknownMode, mode)
}

// Modes は、Build に指定できるモード名を名前順で返します（partialは含みません）。
func (b *Builder) Modes() []string {
	return slices.Clone(b.modes)
}

// Has は、指定されたモードが登録されているかを返します。
func (b *Builder) Has(mode string) bool {
	_, ok := b.modeSet[mode]
	return ok
}

// recordDefinitions は、エントリが定義するテンプレート名を記録し、
// 別のエントリが既に同じ名前を定義していた場合はエラーを返します。
// 全エントリが1つの名前空間を共有するため、検出しないと後勝ちで静かに上書きされます。
//
// 判定にはエントリ単体を解析した結果を使います。root を走査すると
// 他のエントリが定義済みの名前と区別できないためです。
// なお {{template "x"}} による参照は名前空間へエントリを作らないので、
// ここで扱われるのは {{define "x"}} による実際の定義とエントリ自身の名前だけです。
func recordDefinitions(entry, content string, funcs template.FuncMap, definedBy map[string]string) error {
	scratch := template.New(entry)
	if len(funcs) > 0 {
		scratch = scratch.Funcs(funcs)
	}
	if _, err := scratch.Parse(content); err != nil {
		return fmt.Errorf("プロンプト '%s' の解析に失敗: %w", entry, err)
	}

	for _, tmpl := range scratch.Templates() {
		name := tmpl.Name()
		if name == "" || tmpl.Tree == nil {
			continue
		}

		if owner, seen := definedBy[name]; seen && owner != entry {
			return fmt.Errorf("%w: '%s' が '%s' と '%s' の両方で定義されています",
				ErrDuplicateDefinition, name, owner, entry)
		}
		definedBy[name] = entry
	}

	return nil
}

// isPartial は、テンプレート名が partial を指すかを判定します。
// 再帰読み込みで "en/_output" のようなパス形式になる場合を考慮し、末尾の要素で判定します。
// prefix が空文字の場合、partial の判定自体を行いません（全エントリがモードになります）。
func isPartial(name, prefix string) bool {
	if prefix == "" {
		return false
	}
	return strings.HasPrefix(path.Base(name), prefix)
}
