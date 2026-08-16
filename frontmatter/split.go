// Package frontmatter は、プロンプトファイル先頭の front matter を本文から切り離します。
//
// front matter は "---" で挟んだメタデータのブロックで、モードの説明などを
// プロンプト自身に持たせるために使います。本文へ紛れ込むとAIへの指示の先頭に
// メタデータが混ざるため、テンプレートとして登録する前に切り離します。
//
//	files, _ := resource.Load(assets.Prompts, "prompts")
//	bodies, fronts := frontmatter.SplitMap(files)
//	infos, _ := frontmatter.DecodeMap[ModeInfo](fronts, yaml.Unmarshal)
//	builder, _ := prompts.NewBuilder(bodies)
//
// テンプレートの読み込みと組み立てが目的なら、この一連の流れは
// prompts.LoadFS に prompts.WithFrontMatter を渡すことでも行えます。
//
// メタデータの書式はこのパッケージでは解釈しません。解析関数は呼び出し側が
// UnmarshalFunc として渡します（Decode / DecodeMap）。YAMLライブラリの選択と
// 乗り換えを利用側のペースで行えるようにするためで、このパッケージ自体は
// 標準ライブラリだけに依存します。
package frontmatter

import "strings"

// delimiter は front matter ブロックの区切りです。
// 開始・終了ともに、この文字列だけからなる行である必要があります。
const delimiter = "---"

// Split は、先頭の "---\nメタデータ\n---\n" ブロックを本文から切り離します。
// front matter が無い場合、front は空文字、body は content 全体になります。
//
// 終了の区切りは "---" だけからなる行です。"----" のように文字数が違う行は
// 区切りとみなしません。区切り行とその行末の改行だけを取り除くため、
// 区切りの直後に空行がある場合、その空行は本文の一部として残ります。
//
// 返す文字列は、front matter の有無にかかわらず改行を LF に正規化し、
// 先頭のBOMを取り除いたものです。BOMや CRLF はエディタ上で見えないまま
// front matter の判定を外すため、判定前に揃えます。
func Split(content string) (front, body string) {
	normalized := normalize(content)

	opening := delimiter + "\n"
	if !strings.HasPrefix(normalized, opening) {
		return "", normalized
	}

	rest := normalized[len(opening):]

	// 中身のない front matter ブロック。開始行の直後が終了行のケースです。
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
	// 本文が無いファイルでは、終了の区切りがファイル末尾に来ます。
	if strings.HasSuffix(rest, closing) {
		return rest[:len(rest)-len(closing)], ""
	}

	// 終了の区切りが無い。front matter のつもりの記述であっても本文として扱います。
	return "", normalized
}

// SplitMap は、resource.Load が返すマップの各エントリを Split にかけ、
// 本文と front matter をそれぞれのマップに分けて返します。
// キーの集合は入力と同じで、front matter を持たないエントリの front は空文字です。
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

// bom は UTF-8 のバイトオーダーマークです。
const bom = "\ufeff"

// normalize は、判定を妨げる見えない差異（BOM・CRLF）を取り除きます。
func normalize(content string) string {
	return strings.ReplaceAll(strings.TrimPrefix(content, bom), "\r\n", "\n")
}
