# #189 描画性能の調査と改善

- 対象: IT07 / #189 / NFR-01 / DES-IT07 9.1節、DES-UI-29・28、DES-BTL-01・11
- 日付: 2026-11-07
- 基準版: `2cd413c`（#193マージ後。ゲーム本体は#174測定時から変更なし）
- 結果: #174と同じ重い合成条件で、最適化後の3回の測定はいずれも1秒区間のFPSが60以上。実章の初期盤面も別条件で確認した。

## 原因

同じ合成盤面をWindowsで再現すると、最低12.228fpsだった（#174の12.421fpsと同程度）。EnemyRangeLayersとMinimapを各1回求めるCPUベンチマークは72.186ms/op、約3.52MB/op・40,087 allocs/opだった。

[CPUプロファイル](render-performance/profile-before.txt)では、攻撃射程計算`BoardView.AttackRange`が累積64.5%（内側の`rangeCells`は63.4%）、ZOC確認が累積19.6%。大きな値のコピー`runtime.duffcopy`が全体の50.3%を占めた。短い武器の射程を求めるたびに32×32全体を走査し、その内側で武器を値コピーしていた。ZOCの確認でも、隣接していないものを含む全ユニットを値コピーしていた。

また、同じDrawの中で、盤面とミニマップが同じ全敵危険範囲をそれぞれ計算していた。

| CPU計算だけのベンチ | 時間/op | 確保量/op | 確保回数/op |
|---|---:|---:|---:|
| 改善前の盤面範囲＋ミニマップ | 72.186ms | 3,517,576B | 40,087 |
| 射程・コピーのみ改善した中間版の同じ2照会 | 9.490ms | 3,516,805B | 40,089 |
| 最終版の1フレーム内共有 | 4.822ms | 1,980,076B | 21,076 |

最後の行は公開APIをそれぞれ呼ぶのでなく、Game.Drawと同じくDangerを一度求めて両表示へ渡す経路。いずれも同じ合成入力から盤面の層とミニマップを得る計算で、GPU描画の時間は含まない。受入は下記の実際のDrawのFPSで確認する。

## 変更

1. 射程候補の走査を、始点と最大射程で囲む矩形に絞った。元の最小/最大射程の判定、複数武器の合成、y/x順は維持する。巨大な射程は候補矩形の半径を盤面の長辺に制限し、桁あふれを避ける。盤外の始点は従来どおり全マスを調べる。
2. ZOCの読み取りは添字から参照し、全ユニットの値コピーを繰り返さない。
3. 通常戦闘のGame.Drawで危険範囲を一度計算し、盤面とミニマップへ渡す。フレーム間のキャッシュは持たず、次のDrawで現在の盤面から計算する。公開APIと巻き戻しの表示経路は維持する。

ゲームの規則・定義データ・移動費・射程・描画順・見た目を変更していない。要求・設計の原本変更もない。

## 同じ合成条件での測定

基準PC: Intel Core i5-10400F / GTX1660 SUPER / Windows 11 build22631.6199 / Go1.23.5 windows/amd64。GPUドライバ32.0.15.6109。画面倍率1.00、ウィンドウ832×576、VSync有効。Windows CIMは1920×1080・74Hzを報告したが、これはFPSの上限値としては扱わない。比較中に表示設定を変更していない。

合成条件は従来の32×32・自軍7/敵39、全46体にZOCと移動7。全敵範囲、固定2体、カーソルの敵の範囲、装備と指揮の印を同時に描く。戦闘を進めず、2秒慣らして約6秒測る。前後とも全敵の危険マス257、固定[7,8]、カーソルの敵7、カメラ(0,0)。

| 版 | 1秒区間ごとのFPS（約6秒中の完全な5区間） | 最低FPS | 最大Draw間隔 |
|---|---|---:|---:|
| 改善前 | 12.343 / 12.895 / 13.088 / 12.256 / 12.228 | 12.228 | 112.7504ms |
| 射程と値コピーのみ改善した中間版 | 64.760 / 67.260 / 61.764 / 62.489 / 65.857 | 61.764 | 34.7297ms |
| 最終版 1回目 | 79.808 / 91.130 / 81.534 / 82.775 / 94.204 | 79.808 | 33.3028ms |
| 最終版 2回目 | 90.451 / 90.280 / 81.668 / 81.262 / 86.705 | 81.262 | 30.2230ms |
| 最終版 3回目 | 87.286 / 88.273 / 91.868 / 87.112 / 91.468 | 87.112 | 30.1239ms |

この固定条件・測定区間では60fps以上を確認した。OSのスケジューリングなども含む最大Draw間隔は上記のとおりで、全フレーム16.7ms以内を保証する結果ではない。描画するたびに状態が進む場面や、全9章のすべての局面を測ったものでもない。

## 実章との区別

本体データの初期盤面を、seed1・離脱・出撃枠までの生存名簿で作る。成長を仮定した戦闘攻略ではなく、描画用の初期状態。最初の敵にカメラとカーソルを合わせ、最初の敵2体を固定する。全敵範囲・装備等は合成条件と同じく表示する。

| 条件 | 自軍/敵 | 危険マス | 改善前 最低FPS | 最終版 最低FPS |
|---|---:|---:|---:|---:|
| ch01初期 | 5/9 | 194 | 110.780 | 137.208 |
| ch09初期 | 7/29 | 610 | 26.844 | 136.762 |

合成盤面の旧約12fpsを実章全体の速度とは一般化しない。一方、9章の測った初期状態にも旧版の重さがあり、改善後はこの条件でも60fps以上だった。

## 回帰試験

- TC-FR-BTL-01-13: 1×1・細長・5×7・32×32、盤面内外の始点、武器なし・射程1・2・複数射程・最大intの射程を、全マスのマンハッタン距離で独立に求めた列と比較。マス・順序・nilを維持する。改善前から成功する同値性の回帰であり、改善前の性能未達はFPS測定で確認した。
- TC-NFR-01-04: 移動0の弱い敵とボスの手で置いた盤面で、撃破前5マス・撃破後2マス・巻き戻し後5マス・Q切替後0マスを確認。盤面とミニマップで共有しても固定・カーソルの層と順番を保ち、古い範囲を持ち越さない。
- TC-NFR-01-03の測定準備も通常テストで検査し、合成7/39・全員ZOC/Mov7、実章5/9と7/29、固定2体とカーソルの敵、全敵範囲が維持されることを確認する。
- 既存の移動・ZOC・再移動・配置・巻き戻し・範囲表示の試験もそのまま実行する。

## 原記録と再実行

[原ログ](render-performance/)に、before-*（改善前）、after-*（射程/コピーのみの中間版）、final-*（最終版）を保存した。CPUプロファイルはbefore.pprof・core-only.pprof・final.pprof。既存の#174の測定ログは上書きしていない。

```sh
# 改善前のソースを一時ディレクトリへ展開する
mkdir -p /tmp/renkin189-before
git archive 2cd413c | tar -x -C /tmp/renkin189-before
# 同じ測定用fixtureを使う。旧版にない共有処理の追加ベンチだけは除く
python3 - <<'PY'
from pathlib import Path
s = Path('internal/ui/performance_it07_test.go').read_text()
s = s.split('\nfunc BenchmarkIT07SharedFrameRanges(')[0]
Path('/tmp/renkin189-before/internal/ui/performance_it07_test.go').write_text(s)
PY
(cd /tmp/renkin189-before && GOOS=windows go test -c -o /tmp/render-before.exe ./internal/ui)
GOOS=windows go test -c -o /tmp/render-after.exe ./internal/ui
# 下記は基準Windows PCで順に行い、他の重い計測を並行しない
RENKIN_IT07_FPS=1 WSLENV="${WSLENV:+$WSLENV:}RENKIN_IT07_FPS" /tmp/render-before.exe -test.run '^TestIT07FrameTiming$' -test.v
RENKIN_IT07_FPS=1 WSLENV="${WSLENV:+$WSLENV:}RENKIN_IT07_FPS" /tmp/render-after.exe -test.run '^TestIT07FrameTiming$' -test.v
# 実章は両版に同じ RENKIN_IT07_FPS_CH=ch01 または ch09 を渡す
RENKIN_IT07_FPS=1 RENKIN_IT07_FPS_CH=ch09 WSLENV="${WSLENV:+$WSLENV:}RENKIN_IT07_FPS:RENKIN_IT07_FPS_CH" /tmp/render-after.exe -test.run '^TestIT07FrameTiming$' -test.v
# CPU計算だけのベンチ。FPSの代わりにはしない
/tmp/render-before.exe -test.run '^$' -test.bench '^BenchmarkIT07EnemyRangeLayers$' -test.benchtime=3s
/tmp/render-after.exe -test.run '^$' -test.bench '^BenchmarkIT07EnemyRangeLayers$' -test.benchtime=3s
/tmp/render-after.exe -test.run '^$' -test.bench '^BenchmarkIT07SharedFrameRanges$' -test.benchtime=3s
```

## 引き継ぎ

#189の描画性能は、上記の同じ合成条件で60fps以上を確認した。#188のZOC全経路と巻き戻し時の増援計測、#190の9章増援の採否、#175の人の試遊は別Issueとして残す。本変更で解消したとは扱わない。
