// Package audio は新作の効果音と BGM の再生を受け持つ（DES-NG-09）。
//
// 音は media（fs.FS）の audio/se/<名前>.wav と audio/bgm/<名前>.wav に置く。media がなければ全体が無音で、
// ファイルのない音は飛ばす。再生の口は Backend に分け、デバイスがなくても落とさない。
// ui の下の部品なので Ebitengine 系に依存してよいが、core・tactics には依存しない。
//
// Linux で手元ビルドするときは、音の再生（oto）の都合で libasound2-dev が要る。実行時に ALSA のデバイスがなければ、
// 標準エラーに 1 回だけ出して無音で続ける。
package audio

// SE は効果音の名前（DES-NG-09）。ファイルは audio/se/<名前>.wav。
type SE string

// 効果音の一覧。
const (
	SESelect   SE = "select"
	SEConfirm  SE = "confirm"
	SECancel   SE = "cancel"
	SEHit      SE = "hit"
	SEMiss     SE = "miss"
	SECritical SE = "critical"
	SEHeal     SE = "heal"
	SEBlast    SE = "blast"
	SELevelUp  SE = "levelup"
	SEWin      SE = "win"
	SELose     SE = "lose"
	SEStep     SE = "step"
)

// BGM は BGM の名前（DES-NG-09）。ファイルは audio/bgm/<名前>.wav。空の BGM は「止める」を表す。
type BGM string

// BGM の一覧。
const (
	BGMNone    BGM = ""
	BGMTitle   BGM = "title"
	BGMStory   BGM = "story"
	BGMPrepare BGM = "prepare"
	BGMBattle  BGM = "battle"
)

// AllSE は効果音の全部（定数との対応のテストに使う）。
func AllSE() []SE {
	return []SE{SESelect, SEConfirm, SECancel, SEHit, SEMiss, SECritical, SEHeal, SEBlast, SELevelUp, SEWin, SELose, SEStep}
}

// AllBGM は BGM の全部（BGMNone を除く）。
func AllBGM() []BGM { return []BGM{BGMTitle, BGMStory, BGMPrepare, BGMBattle} }

// Path は media の下のファイルの道筋。
func (s SE) Path() string { return "audio/se/" + string(s) + ".wav" }

// Path は media の下のファイルの道筋。
func (b BGM) Path() string { return "audio/bgm/" + string(b) + ".wav" }
