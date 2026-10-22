package core

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// Rule は倒れた仲間の扱い（DES-CMP-01）。
type Rule int

const (
	// Withdraw は次の章で復帰する離脱。
	Withdraw Rule = iota
	// Permadeath は以後出撃できないロスト。
	Permadeath
)

// キャンペーン操作の誤り。errors.Is で見分ける（DES-CMP-08）。
var (
	ErrInvalidRule       = errors.New("core: 離脱・ロストの選択が正しくありません")
	ErrCleared           = errors.New("core: キャンペーンはクリア済みです")
	ErrInvalidMember     = errors.New("core: 名簿にいないか戦死している仲間です")
	ErrNoStock           = errors.New("core: 在庫にアイテムがありません")
	ErrHandFull          = errors.New("core: 手持ちの上限は 5 個です")
	ErrNoHandItem        = errors.New("core: 手持ちにアイテムがありません")
	ErrInvalidDeployment = errors.New("core: 出撃の選び方が正しくありません")
	ErrInvalidSortie     = errors.New("core: この章の未反映の出撃ではありません")
	ErrNotWon            = errors.New("core: 戦闘に勝っていません")
)

// Campaign は章の進行と名簿・在庫を持つ。目録はセーブに含めない（DES-CMP-01）。
type Campaign struct {
	Rule    Rule
	Chapter int
	Cleared bool
	Roster  []Member
	Stock   []defs.ItemCount
	cat     *defs.Catalog
}

// Member は章をまたいで残る仲間の状態。Stats.HP は最大 HP（DES-CMP-01）。
type Member struct {
	Character string
	Level     int
	Exp       int
	Stats     tactics.Stats
	Items     []defs.ItemCount
	Alive     bool
}

// Sortie は戦闘と出撃順の名簿の添字を持つ（DES-CMP-05）。
// Battle と Members は参照用。出撃時の対応は内部に写しを保持する。
type Sortie struct {
	Battle   *tactics.Battle
	Members  []int
	campaign *Campaign
	chapter  int
	finished bool
	members  []int
	before   []Member
}

// ChapterResult は終えた章の結果。Members は出撃順（DES-CMP-06）。
type ChapterResult struct {
	Chapter string
	Members []MemberResult
	Reward  []defs.ItemCount
	Cleared bool
}

// MemberResult は出撃した仲間の成長と離脱・戦死（DES-CMP-06）。
type MemberResult struct {
	Member      int
	LevelBefore int
	LevelAfter  int
	ExpBefore   int
	ExpAfter    int
	Gains       tactics.Stats
	Fallen      bool
	Dead        bool
}

// NewCampaign は指定されたルールで最初の章に入る（DES-CMP-03）。
// cat は defload で検証済みの目録を受け取る。
func NewCampaign(cat *defs.Catalog, rule Rule) (*Campaign, error) {
	if rule != Withdraw && rule != Permadeath {
		return nil, ErrInvalidRule
	}
	c := &Campaign{Rule: rule, cat: cat}
	for _, id := range cat.Game.Party {
		m, err := newMember(cat, id)
		if err != nil {
			return nil, err
		}
		c.Roster = append(c.Roster, m)
	}
	for _, ic := range cat.Game.Stock {
		c.Stock = changeItem(c.Stock, ic.Item, ic.Count)
	}
	c.sortStock()
	if err := c.enterChapter(); err != nil {
		return nil, err
	}
	return c, nil
}

// newMember はキャラの加入時の状態を作る。手持ちは定義から写す（DES-CMP-03）。
func newMember(cat *defs.Catalog, id string) (Member, error) {
	ch, ok := cat.Character(id)
	if !ok {
		return Member{}, fmt.Errorf("core: キャラ %q がありません", id)
	}
	cls, ok := cat.Class(ch.Class)
	if !ok {
		return Member{}, fmt.Errorf("core: キャラ %q のクラス %q がありません", id, ch.Class)
	}
	return Member{Character: id, Level: ch.Level, Stats: addStats(cls.Base, ch.Bonus), Items: append([]defs.ItemCount(nil), ch.Items...), Alive: true}, nil
}

// enterChapter はまだ名簿にいない加入者を定義順に足す。戦死者も再加入しない（DES-CMP-03）。
func (c *Campaign) enterChapter() error {
	chapters := c.cat.Chapters()
	if c.Chapter < 0 || c.Chapter >= len(chapters) {
		return fmt.Errorf("core: 章 %d がありません", c.Chapter)
	}
	for _, id := range chapters[c.Chapter].Joins {
		found := false
		for _, m := range c.Roster {
			if m.Character == id {
				found = true
				break
			}
		}
		if found {
			continue
		}
		m, err := newMember(c.cat, id)
		if err != nil {
			return err
		}
		c.Roster = append(c.Roster, m)
	}
	return nil
}

// TakeItem は在庫の 1 個を生存している仲間へ渡す（DES-CMP-04、DES-CMP-08）。
func (c *Campaign) TakeItem(member int, item string) error {
	if err := c.prepareMember(member); err != nil {
		return err
	}
	if itemCount(c.Stock, item) < 1 {
		return ErrNoStock
	}
	total := 0
	for _, ic := range c.Roster[member].Items {
		total += ic.Count
	}
	if total >= 5 {
		return ErrHandFull
	}
	c.Roster[member].Items = changeItem(c.Roster[member].Items, item, 1)
	c.Stock = changeItem(c.Stock, item, -1)
	c.sortStock()
	return nil
}

// StoreItem は生存している仲間の手持ち 1 個を在庫に戻す（DES-CMP-04、DES-CMP-08）。
func (c *Campaign) StoreItem(member int, item string) error {
	if err := c.prepareMember(member); err != nil {
		return err
	}
	if itemCount(c.Roster[member].Items, item) < 1 {
		return ErrNoHandItem
	}
	c.Roster[member].Items = changeItem(c.Roster[member].Items, item, -1)
	c.Stock = changeItem(c.Stock, item, 1)
	c.sortStock()
	return nil
}

// prepareMember は準備の共通の検証を順に行う（DES-CMP-04）。
func (c *Campaign) prepareMember(member int) error {
	if c.Cleared {
		return ErrCleared
	}
	if member < 0 || member >= len(c.Roster) || !c.Roster[member].Alive {
		return ErrInvalidMember
	}
	return nil
}

// itemCount は指定された ID の個数を返す（DES-CMP-04）。
func itemCount(items []defs.ItemCount, id string) int {
	for _, ic := range items {
		if ic.Item == id {
			return ic.Count
		}
	}
	return 0
}

// changeItem は個数を加減し、0 の要素を消す。新しい ID は末尾に足す（DES-CMP-04）。
// 常に写しを返すので、失敗時の巻き戻しや結果の保持と配列を共有しない。
func changeItem(items []defs.ItemCount, id string, delta int) []defs.ItemCount {
	out := append([]defs.ItemCount(nil), items...)
	for i := range out {
		if out[i].Item == id {
			out[i].Count += delta
			if out[i].Count == 0 {
				out = append(out[:i], out[i+1:]...)
			}
			return out
		}
	}
	return append(out, defs.ItemCount{Item: id, Count: delta})
}

// sortStock は在庫を ID 順にする（DES-CMP-01）。
func (c *Campaign) sortStock() {
	sort.Slice(c.Stock, func(i, j int) bool { return c.Stock[i].Item < c.Stock[j].Item })
}

// StartBattle は名簿を変えず、選択順に出撃の枠へ配置する（DES-CMP-05、DES-DEP-01）。
func (c *Campaign) StartBattle(members []int, seed uint64) (*Sortie, []tactics.Event, error) {
	slots := make([]int, len(members))
	for i := range slots {
		slots[i] = i
	}
	return c.StartBattleAt(members, slots, seed)
}

// StartBattleAt は出撃順を保ち、各仲間を指定した枠に配置する（DES-DEP-01）。
// members[0] は主人公とし、slots[i] は members[i] を置くマップの枠の添字。
// 主人公も空き枠へ移せる。検証・組み立てに失敗しても Campaign と入力は変えない。
func (c *Campaign) StartBattleAt(members []int, slots []int, seed uint64) (*Sortie, []tactics.Event, error) {
	if c.Cleared {
		return nil, nil, ErrCleared
	}
	chapter := c.cat.Chapters()[c.Chapter]
	m, ok := c.cat.Map(chapter.Map)
	if !ok {
		return nil, nil, fmt.Errorf("core: マップ %q がありません", chapter.Map)
	}
	if len(members) < 1 || len(members) > len(m.Slots) {
		return nil, nil, fmt.Errorf("%w: 人数は 1 人以上 %d 人以下です", ErrInvalidDeployment, len(m.Slots))
	}
	seen := map[int]bool{}
	for _, i := range members {
		if i < 0 || i >= len(c.Roster) || !c.Roster[i].Alive || seen[i] {
			return nil, nil, fmt.Errorf("%w: 名簿の添字 %d は範囲外・戦死・重複のいずれかです", ErrInvalidDeployment, i)
		}
		seen[i] = true
	}
	lord, ok := c.cat.Character(c.Roster[members[0]].Character)
	if !ok || !lord.Lord {
		return nil, nil, fmt.Errorf("%w: 先頭は主人公でなければなりません", ErrInvalidDeployment)
	}
	if len(slots) != len(members) {
		return nil, nil, fmt.Errorf("%w: 枠と出撃者の長さが違います", ErrInvalidDeployment)
	}
	for i, slot := range slots {
		if slot < 0 || slot >= len(m.Slots) {
			return nil, nil, fmt.Errorf("%w: slots[%d] の枠 %d は範囲外です", ErrInvalidDeployment, i, slot)
		}
	}
	used := make(map[int]bool, len(slots))
	for _, slot := range slots {
		if used[slot] {
			return nil, nil, fmt.Errorf("%w: 枠 %d が重複しています", ErrInvalidDeployment, slot)
		}
		used[slot] = true
	}
	var allies []tactics.UnitSetup
	var before []Member
	for i, index := range members {
		member := c.Roster[index]
		u, err := allyUnit(c.cat, member.Character, m.Slots[slots[i]])
		if err != nil {
			return nil, nil, err
		}
		u.Level = member.Level
		u.Exp = member.Exp
		u.Stats = member.Stats
		u.Items, err = convertItems(c.cat, member.Items)
		if err != nil {
			return nil, nil, err
		}
		allies = append(allies, u)
		member.Items = append([]defs.ItemCount(nil), member.Items...)
		before = append(before, member)
	}
	setup, err := setupWithAllies(c.cat, m, allies)
	if err != nil {
		return nil, nil, err
	}
	b, events, err := tactics.New(setup, seed)
	if err != nil {
		return nil, nil, err
	}
	return &Sortie{Battle: b, Members: append([]int(nil), members...), members: append([]int(nil), members...), before: before, campaign: c, chapter: c.Chapter}, events, nil
}

// FinishChapter は勝利の成長・手持ち・報酬を反映し、次の章に入る（DES-CMP-06〜08）。
// 加入時に誤りがあっても状態を変えないよう、全体の写しに反映してから置き換える。
func (c *Campaign) FinishChapter(s *Sortie) (ChapterResult, error) {
	if s == nil || s.campaign != c || s.chapter != c.Chapter || s.finished || s.Battle == nil {
		return ChapterResult{}, ErrInvalidSortie
	}
	if s.Battle.Result != tactics.Won {
		return ChapterResult{}, ErrNotWon
	}
	next := *c
	next.Roster = append([]Member(nil), c.Roster...)
	next.Stock = append([]defs.ItemCount(nil), c.Stock...)
	chapter := c.cat.Chapters()[c.Chapter]
	result := ChapterResult{Chapter: chapter.ID, Reward: append([]defs.ItemCount(nil), chapter.Reward...)}
	units := s.Battle.Units()
	for i, index := range s.members {
		u := units[i]
		before := s.before[i]
		member := &next.Roster[index]
		dead := !u.Alive && c.Rule == Permadeath
		result.Members = append(result.Members, MemberResult{Member: index, LevelBefore: before.Level, LevelAfter: u.Level, ExpBefore: before.Exp, ExpAfter: u.Exp, Gains: statsDifference(u.Stats, before.Stats), Fallen: !u.Alive, Dead: dead})
		member.Level = u.Level
		member.Exp = u.Exp
		member.Stats = u.Stats
		member.Alive = !dead
		member.Items = nil
		if !dead {
			for _, ic := range u.Items {
				member.Items = append(member.Items, defs.ItemCount{Item: ic.Item.ID, Count: ic.Count})
			}
		}
	}
	for _, ic := range chapter.Reward {
		next.Stock = changeItem(next.Stock, ic.Item, ic.Count)
	}
	next.sortStock()
	next.Chapter++
	if next.Chapter == len(c.cat.Chapters()) {
		next.Cleared = true
	} else if err := next.enterChapter(); err != nil {
		return ChapterResult{}, err
	}
	result.Cleared = next.Cleared
	*c = next
	s.finished = true
	return result, nil
}

// statsDifference は章の前後の能力の差を求める（DES-CMP-06）。
func statsDifference(after, before tactics.Stats) tactics.Stats {
	return tactics.Stats{HP: after.HP - before.HP, Str: after.Str - before.Str, Skl: after.Skl - before.Skl, Spd: after.Spd - before.Spd, Def: after.Def - before.Def, Mov: after.Mov - before.Mov}
}
