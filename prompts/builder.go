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

	// ErrLoadOnlyOption は、読み込み時のみ有効なオプションが NewBuilder に
	// 渡された場合に返されます。黙って無視すると、指定したつもりの絞り込みが
	// 効いていないことに気付けないためです。
	ErrLoadOnlyOption = errors.New("LoadFS でのみ有効なオプションです")
)

// Builder は、読み込み済みのテンプレート一式とモード選択の規則を保持します。
// すべてのテンプレートは1つの名前空間へ関連付けて登録されるため、
// モード本文から partial を {{template "_name" .}} で参照できます。
//
// 構築後の Builder は不変で、すべての公開メソッドを複数の goroutine から
// 同時に呼び出せます。HTTPハンドラのように並行して呼ばれる場所では、
// 起動時に一度構築して使い回してください。
type Builder struct {
	root        *template.Template
	modes       []string
	modeSet     map[string]struct{}
	defaultMode string
	// fronts は WithFrontMatter で切り離した front matter です（エントリ名 -> 生の文字列）。
	fronts map[string]string
}

// NewBuilder は、モード名をキーとするテンプレートのマップから Builder を構築します。
// DefaultPartialPrefix で始まるエントリは partial として扱われます（IsPartial を参照）。
//
// 読み込みに関するオプションは LoadFS 専用です。渡された場合は ErrLoadOnlyOption を返します。
func NewBuilder(templates map[string]string, opts ...Option) (*Builder, error) {
	cfg := newConfig(opts...)

	if len(cfg.loadOnly) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrLoadOnlyOption, strings.Join(cfg.loadOnly, ", "))
	}

	return newBuilder(templates, cfg)
}

// newBuilder は、適用済みの設定から Builder を構築します。
// LoadFS は読み込み専用オプションを自身で消費するため、NewBuilder の検査を通さずここを呼びます。
func newBuilder(templates map[string]string, cfg *config) (*Builder, error) {
	if len(templates) == 0 {
		return nil, ErrEmptyTemplates
	}

	// missingkey=error は root へ設定します。オプションは関連テンプレート間で
	// 共有されるため、これだけで partial を含む全エントリへ効きます。
	root := template.New("").Option("missingkey=error")
	if len(cfg.funcs) > 0 {
		root = root.Funcs(cfg.funcs)
	}

	modes := make([]string, 0, len(templates))
	modeSet := make(map[string]struct{}, len(templates))
	// テンプレート名 -> それを定義したエントリ名（重複定義の検出用。parseEntry を参照）。
	definedBy := make(map[string]string, len(templates))

	// エラー報告の順序を安定させるため、名前順に処理します。
	for _, name := range slices.Sorted(maps.Keys(templates)) {
		content := templates[name]
		if content == "" {
			return nil, fmt.Errorf("プロンプトテンプレート '%s' の読み込みに失敗しました: 内容が空です", name)
		}
		// 空かどうかを見てから取り除きます。改行だけの partial は、これまでどおり
		// 「何も出力しない partial」として登録されます（空の内容とは別の誤りです）。
		if cfg.trimPartials && IsPartial(name, cfg.partialPrefix) {
			content = strings.TrimRight(content, "\r\n")
		}

		parsed, err := parseEntry(name, content, cfg.funcs, definedBy)
		if err != nil {
			return nil, err
		}
		if err := attach(root, parsed); err != nil {
			return nil, fmt.Errorf("プロンプト '%s' の登録に失敗: %w", name, err)
		}

		if !IsPartial(name, cfg.partialPrefix) {
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

// Build は、指定されたモードのテンプレートを data で実行します。
// WithDefaultMode が指定されている場合、未登録のモード（空文字を含む）は既定モードへ委ねられます。
//
// テンプレートが参照するフィールドが data に無い場合はエラーになります
// （"<no value>" として黙って出力されることはありません）。
// data の中身そのものの妥当性は検証しないため、値の確認は呼び出し側の責務です。
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
// 既定モード（WithDefaultMode）へのフォールバックは考慮せず、実際の登録有無を返します。
func (b *Builder) Has(mode string) bool {
	_, ok := b.modeSet[mode]
	return ok
}

// FrontMatter は、指定したエントリの front matter を切り離したままの文字列で返します。
// LoadFS に WithFrontMatter を指定した場合のみ値を持ちます。
// front matter を持たないエントリと未知のエントリはどちらも空文字を返します。
//
// 書式の解釈はこのパッケージでは行いません。frontmatter.Decode へ渡してください。
func (b *Builder) FrontMatter(name string) string {
	return b.fronts[name]
}

// FrontMatters は、切り離した front matter をエントリ名をキーとするマップで返します
// （partial を含みます）。frontmatter.DecodeMap へそのまま渡せます。
// 呼び出し側が書き換えても Builder には影響しません。
func (b *Builder) FrontMatters() map[string]string {
	return maps.Clone(b.fronts)
}

// parseEntry は、エントリ単体を解析し、そのエントリが定義するテンプレート名を記録します。
// 別のエントリが既に同じ名前を定義していた場合はエラーを返します。
// 全エントリが1つの名前空間を共有するため、検出しないと後勝ちで静かに上書きされます。
//
// 判定を隔離した解析結果で行うのは、root を走査すると他のエントリが定義済みの名前と
// 区別できないためです。なお {{template "x"}} による参照は構文木を持たないエントリを
// 作るため、ここで扱われるのは {{define "x"}} による実際の定義とエントリ自身の名前だけです。
func parseEntry(entry, content string, funcs template.FuncMap, definedBy map[string]string) (*template.Template, error) {
	scratch := template.New(entry)
	if len(funcs) > 0 {
		scratch = scratch.Funcs(funcs)
	}
	if _, err := scratch.Parse(content); err != nil {
		return nil, fmt.Errorf("プロンプト '%s' の解析に失敗: %w", entry, err)
	}

	for _, tmpl := range scratch.Templates() {
		name := tmpl.Name()
		if name == "" || tmpl.Tree == nil {
			continue
		}

		if owner, seen := definedBy[name]; seen && owner != entry {
			return nil, fmt.Errorf("%w: '%s' が '%s' と '%s' の両方で定義されています",
				ErrDuplicateDefinition, name, owner, entry)
		}
		definedBy[name] = entry
	}

	return scratch, nil
}

// attach は、隔離して解析した構文木を共有の名前空間へ移します。
// 構文木を渡すだけなので、同じ内容をもう一度解析することはありません。
// 構文木を持たないエントリ（{{template "x"}} による未定義への参照）は、
// 定義済みの構文木を消さないよう対象外にします。
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

// IsPartial は、テンプレート名が partial（モードではなく参照専用の部品）を指すかを判定します。
// 再帰読み込みで "en/_output" のようなパス形式になる場合を考慮し、末尾の要素で判定します。
// prefix が空文字の場合、partial の判定自体を行いません（全エントリがモードになります）。
//
// Builder は同じ規則で Modes() と Build() の対象を決めます。読み込んだマップを
// Builder へ渡す前に自前で選り分ける場合は、判定を二重に書かずこの関数を使ってください。
//
//	if prompts.IsPartial(name, prompts.DefaultPartialPrefix) { continue }
func IsPartial(name, prefix string) bool {
	if prefix == "" {
		return false
	}
	return strings.HasPrefix(path.Base(name), prefix)
}
