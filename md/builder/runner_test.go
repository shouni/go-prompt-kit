package builder

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		options   []Option
		wantHTML  bool
		wantWraps bool
		wantMode  string
		wantErr   bool
	}{
		{
			name:      "デフォルト設定で生成",
			options:   nil,
			wantHTML:  false,
			wantWraps: false,
			wantMode:  htmlMode,
		},
		{
			name: "オプションでUnsafeHTMLとHardWrapsを有効化",
			options: []Option{
				WithEnableUnsafeHTML(true),
				WithEnableHardWraps(true),
			},
			wantHTML:  true,
			wantWraps: true,
			wantMode:  htmlMode,
		},
		{
			name: "HTMLモードを明示的に指定",
			options: []Option{
				WithHTMLMode(),
			},
			wantHTML:  false,
			wantWraps: false,
			wantMode:  htmlMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// renderer.NewRenderer() が embed.FS を参照するため、
			// テスト実行環境でアセットが読み込める状態である必要があります。
			b, err := New(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if b.config.enableUnsafeHTML != tt.wantHTML {
				t.Errorf("config.enableUnsafeHTML = %v, want %v", b.config.enableUnsafeHTML, tt.wantHTML)
			}
			if b.config.enableHardWraps != tt.wantWraps {
				t.Errorf("config.enableHardWraps = %v, want %v", b.config.enableHardWraps, tt.wantWraps)
			}
			if b.config.mode != tt.wantMode {
				t.Errorf("config.mode = %v, want %v", b.config.mode, tt.wantMode)
			}
		})
	}
}

func TestBuilder_BuildRunner(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{
			name:    "正常系: htmlモード",
			mode:    htmlMode,
			wantErr: false,
		},
		{
			name:    "正常系: 空文字（デフォルト挙動）",
			mode:    "",
			wantErr: false,
		},
		{
			name:    "異常系: 未サポートのモード",
			mode:    "unsupported_mode",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				config: config{mode: tt.mode},
				// インターフェースを満たす nil を代入
				converter: nil,
				renderer:  nil,
			}

			got, err := b.BuildRunner()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildRunner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("BuildRunner() returned nil runner without error")
			}
		})
	}
}
