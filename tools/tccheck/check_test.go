package main

import (
	"os"
	"strings"
	"testing"
)

// テストケース ID の整合検査を、testdata の小さなリポジトリで確かめる（QA-02）。
func TestCheck(t *testing.T) {
	cases := []struct {
		name       string
		dir        string
		wantErrors []string
		wantWarn   int
	}{
		{"TC-QA-02-01 範囲表記と結果の表、テストコードの重複を正しく扱う", "ok", nil, 1},
		{"TC-QA-02-02 テスト仕様書の重複を検出する", "dup", []string{"テストケース ID TC-FR-AAA-01-01 が重複"}, 1},
		{"TC-QA-02-03 テスト仕様書にない ID を検出する", "undefined", []string{
			"テストコードの TC-FR-BBB-02-01 がテスト仕様書にありません",
			"トレーサビリティの TC-FR-AAA-01-02 がテスト仕様書にありません",
			"トレーサビリティの TC-FR-AAA-01-03 がテスト仕様書にありません",
		}, 0},
		{"TC-QA-02-04 トレーサビリティにない ID を検出する", "untraced", []string{": TC-FR-AAA-01-02 がトレーサビリティマトリクスにありません"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Check(os.DirFS("testdata/" + tc.dir))
			if len(r.Errors) != len(tc.wantErrors) {
				t.Fatalf("誤りの数 = %d、期待値 = %d: %v", len(r.Errors), len(tc.wantErrors), r.Errors)
			}
			for _, want := range tc.wantErrors {
				found := false
				for _, e := range r.Errors {
					found = found || strings.Contains(e, want)
				}
				if !found {
					t.Errorf("%q を含む誤りがない: %v", want, r.Errors)
				}
			}
			if len(r.Warnings) != tc.wantWarn {
				t.Errorf("警告の数 = %d、期待値 = %d: %v", len(r.Warnings), tc.wantWarn, r.Warnings)
			}
		})
	}
}
