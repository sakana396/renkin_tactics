package tactics

import "errors"

// 誤りの値（DES-BTL-08、DES-ERR-01）。errors.Is で見分ける。
var (
	// ErrBattleOver は、戦闘がもう終わっていることを表す。
	ErrBattleOver = errors.New("tactics: 戦闘はもう終わっています")
	// ErrNotPlayerPhase は、自軍の番ではないことを表す。
	ErrNotPlayerPhase = errors.New("tactics: 自軍の番ではありません")
	// ErrInvalidUnit は、ユニットがいない、倒れている、または自軍でないことを表す。
	ErrInvalidUnit = errors.New("tactics: そのユニットは動かせません（いない、倒れている、または自軍でない）")
	// ErrAlreadyActed は、ユニットがすでに行動したことを表す。
	ErrAlreadyActed = errors.New("tactics: そのユニットはすでに行動しました")
	// ErrUnreachable は、そのマスに移動できないことを表す。
	ErrUnreachable = errors.New("tactics: そのマスには移動できません")
	// ErrNoTarget は、狙う相手がいない（いない、倒れている、味方である）ことを表す。
	ErrNoTarget = errors.New("tactics: 狙う相手がいません")
	// ErrOutOfRange は、相手が武器の射程に入っていないことを表す。
	ErrOutOfRange = errors.New("tactics: 相手が武器の射程に入っていません")
	// ErrNoItem は、そのアイテムがないことを表す。
	ErrNoItem = errors.New("tactics: そのアイテムはありません")
	// ErrFullHP は、HP が最大なので回復薬を使えないことを表す。
	ErrFullHP = errors.New("tactics: HP が最大なので使えません")
	// ErrInvalidAction は、Action が定められた値でないこと、
	// または Forecast に Attack でないコマンドを渡したことを表す。
	ErrInvalidAction = errors.New("tactics: 行動の種類が正しくありません")
	// ErrInvalidSetup は、Setup が正しくないことを表す（New だけ）。
	ErrInvalidSetup = errors.New("tactics: 戦闘の入力が正しくありません")
)

// errNotImplemented は、まだ作っていない行動（UseItem）を表す。
// アイテム（#9）の Issue で取り除く。
var errNotImplemented = errors.New("tactics: この行動はまだ作っていません")
