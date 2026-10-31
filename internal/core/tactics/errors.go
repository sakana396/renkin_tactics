package tactics

import "errors"

// 誤りの値（DES-BTL-08、DES-ERR-01）。errors.Is で見分ける。
var (
	// 巻き戻しと再開の誤り（DES-RWD-02・04）。
	ErrNoRewinds       = errors.New("tactics: 巻き戻しの残り回数がありません")
	ErrInvalidRewind   = errors.New("tactics: 巻き戻し先が正しくありません")
	ErrResumeRequired  = errors.New("tactics: 番の処理の再開が必要です")
	ErrNoPendingResume = errors.New("tactics: 再開する処理がありません")
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
	// ErrNoTarget は、狙う相手がいないことを表す。
	// 攻撃では、いない、倒れている、味方である。回復では、いない、倒れている、敵である、自分である（DES-HEL-01）。
	ErrNoTarget = errors.New("tactics: 狙う相手がいません")
	// ErrOutOfRange は、相手が武器の射程に入っていないことを表す。
	ErrOutOfRange = errors.New("tactics: 相手が武器の射程に入っていません")
	// ErrNoItem は、そのアイテムがないことを表す。
	ErrNoItem = errors.New("tactics: そのアイテムはありません")
	// ErrFullHP は、HP が最大なので、回復薬を使えない（UseItem）、
	// または回復できない（Heal。相手の HP）ことを表す（DES-HEL-01）。
	ErrFullHP = errors.New("tactics: HP が最大なので使えません")
	// ErrInvalidAction は、Action が定められた値でないこと、
	// または Forecast に Attack でないコマンドを、HealForecast に Heal でないコマンドを渡したことを表す（DES-HEL-02）。
	ErrInvalidAction = errors.New("tactics: 行動の種類が正しくありません")
	// ErrCannotAttack は、杖のユニットは攻撃できないことを表す（DES-HEL-01）。
	ErrCannotAttack = errors.New("tactics: 杖のユニットは攻撃できません")
	// ErrCannotHeal は、杖を持っていないので回復できないことを表す（DES-HEL-01）。
	ErrCannotHeal = errors.New("tactics: 杖を持っていないので回復できません")
	// ErrNoUses は、杖の使用回数が残っていないことを表す（DES-HEL-01、E-33）。
	ErrNoUses = errors.New("tactics: 杖の使用回数が残っていません")
	// ErrCantoPending は、再移動を待っている間に、ほかのユニットの行動や番の終わりを求めたことを表す（DES-BTL-10）。
	ErrCantoPending = errors.New("tactics: 再移動を待っています（再移動するか、その場で終えてください）")
	// ErrNoCantoPending は、待っている再移動がないのに CantoMove・CantoEnd を求めたことを表す（DES-BTL-10）。
	ErrNoCantoPending = errors.New("tactics: 待っている再移動がありません")
	// ErrInvalidSetup は、Setup が正しくないことを表す（New だけ）。
	ErrInvalidSetup = errors.New("tactics: 戦闘の入力が正しくありません")
)
