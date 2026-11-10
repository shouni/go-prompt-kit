package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
)

// config holds the settings for building a Converter. It accumulates the options for goldmark.New
// by kind and applies them once inside New.
type config struct {
	rendererOptions []renderer.Option
	parserOptions   []parser.Option
	extensions      []goldmark.Extender
	goldmarkOptions []goldmark.Option
}

// Option is a function that configures a Converter.
type Option func(*config)

// WithUnsafeHTML allows "unsafe" HTML output in the goldmark renderer. When enabled, raw HTML in
// the Markdown is output as is, so do not use it with untrusted input.
func WithUnsafeHTML(enable bool) Option {
	return func(c *config) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithUnsafe())
		}
	}
}

// WithHardWraps converts plain line breaks in the Markdown into <br> tags.
func WithHardWraps(enable bool) Option {
	return func(c *config) {
		if enable {
			c.rendererOptions = append(c.rendererOptions, html.WithHardWraps())
		}
	}
}

// WithAutoHeadingID adds id attributes to headings automatically. Enable it to link to anchors from
// a table of contents.
func WithAutoHeadingID(enable bool) Option {
	return func(c *config) {
		if enable {
			c.parserOptions = append(c.parserOptions, parser.WithAutoHeadingID())
		}
	}
}

// WithTypographer replaces quotes, dashes and the like with typographic symbols.
func WithTypographer(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.Typographer)
		}
	}
}

// WithCJK enables line-break handling for East Asian languages such as Japanese.
//
// In Markdown a line break inside a paragraph (a soft line break) stays in the output HTML, and a
// browser renders it as a space. That is correct for languages that separate words with spaces, but
// in Japanese it puts a space in the middle of a sentence.
//
//	これは日本語の文章で、
//	途中で改行しています。
//
//	off: <p>これは日本語の文章で、\n途中で改行しています。</p>  → displayed as "、 途中で"
//	on:  <p>これは日本語の文章で、途中で改行しています。</p>
//
// Only a break with East Asian wide characters on both sides is removed, so in a document mixing
// Japanese and English the separators on the English side are kept.
//
// htmldoc's default lang is "ja-jp", but this is off by default, consistent with the other
// extension options.
func WithCJK(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.CJK)
		}
	}
}

// WithFootnotes enables footnote syntax ([^1]).
func WithFootnotes(enable bool) Option {
	return func(c *config) {
		if enable {
			c.extensions = append(c.extensions, extension.Footnote)
		}
	}
}

// WithGoldmarkOptions is an extension point for passing settings the options above cannot express
// directly as goldmark.Option.
func WithGoldmarkOptions(opts ...goldmark.Option) Option {
	return func(c *config) {
		c.goldmarkOptions = append(c.goldmarkOptions, opts...)
	}
}
