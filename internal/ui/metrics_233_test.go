package ui

import (
	"image"
	"testing"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name   string
		cell   int
		want   Metrics
		offset image.Point
	}{
		{"TC-DES-NG-10-35 既定", 32, Metrics{Cell: 32, Map: 512, ScreenW: 832, ScreenH: 576,
			forecastCandidates:   [4]image.Rectangle{image.Rect(0, 8, 416, 248), image.Rect(96, 8, 512, 248), image.Rect(0, 264, 416, 504), image.Rect(96, 264, 512, 504)},
			forecastFallbackRect: image.Rect(512, 0, 832, 360), summaryX: 520, summaryPortraitX: 760, summaryNarrow: 232,
			detailButtonRect: image.Rect(672, 304, 744, 336), deployDetailButtonRect: image.Rect(752, 336, 824, 356), detailBottom: 504,
			rewindLeft: 520, rewindRight: 824, cameraGuideY: 560, hintRewindY: 560, minimapBoxX: 608,
			equipMarkSize: 10, equipMarkOffset: 22, unitHPBarRight: 21, zocEnemyCornerY: 24,
			overviewButtonRect: image.Rect(744, 472, 824, 504), nextAllyButtonRect: image.Rect(704, 520, 824, 544), rangeStatusRect: image.Rect(520, 304, 664, 336), rangeClearRect: image.Rect(752, 304, 824, 336)}, image.Pt(0, 0)},
		{"TC-DES-NG-10-36 高解像度", 48, Metrics{Cell: 48, Map: 768, ScreenW: 1088, ScreenH: 832,
			forecastCandidates:   [4]image.Rectangle{image.Rect(0, 8, 416, 248), image.Rect(352, 8, 768, 248), image.Rect(0, 520, 416, 760), image.Rect(352, 520, 768, 760)},
			forecastFallbackRect: image.Rect(768, 0, 1088, 360), summaryX: 776, summaryPortraitX: 1016, summaryNarrow: 232,
			detailButtonRect: image.Rect(928, 304, 1000, 336), deployDetailButtonRect: image.Rect(1008, 336, 1080, 356), detailBottom: 760,
			rewindLeft: 776, rewindRight: 1080, cameraGuideY: 816, hintRewindY: 816, minimapBoxX: 864,
			equipMarkSize: 15, equipMarkOffset: 33, unitHPBarRight: 32, zocEnemyCornerY: 36,
			overviewButtonRect: image.Rect(1000, 472, 1080, 504), nextAllyButtonRect: image.Rect(960, 776, 1080, 800), rangeStatusRect: image.Rect(776, 304, 920, 336), rangeClearRect: image.Rect(1008, 304, 1080, 336)}, image.Pt(128, 128)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NewMetrics(tc.cell)
			if got != tc.want {
				t.Errorf("NewMetrics(%d) = %+v、期待値 %+v", tc.cell, got, tc.want)
			}
			if got.sceneOffset() != tc.offset {
				t.Errorf("sceneOffset = %v、期待値 %v", got.sceneOffset(), tc.offset)
			}
			if got.scaled(32) != tc.cell || got.boardShift() != tc.want.Map-512 {
				t.Error("比例または盤面のずれが一致しない")
			}
		})
	}
}
