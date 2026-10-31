package core_test

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sakana396/renkin_tactics/data"
	"github.com/sakana396/renkin_tactics/internal/core"
	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defload"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// loadBase は本体のデータ（data）を読み込んだ目録を返す。
func loadBase(t *testing.T) *defs.Catalog {
	t.Helper()
	cat, err := defload.Load(data.Base())
	if err != nil {
		t.Fatalf("本体のデータに誤りがある:\n%v", err)
	}
	return cat
}

// newCh01 は本体のデータから ch01 の戦闘を作る。
func newCh01(t *testing.T) (*tactics.Battle, []tactics.Event) {
	t.Helper()
	b, events, err := core.NewBattle(loadBase(t), "ch01", 1)
	if err != nil {
		t.Fatalf("NewBattle: %v", err)
	}
	return b, events
}

var (
	potion  = tactics.Item{ID: "potion", Name: "回復薬", Heal: 10}
	sword   = tactics.Weapon{Name: "鉄の剣", Type: tactics.Sword, Weight: 4, Strikes: 1, Might: 5, Hit: 90, MinRange: 1, MaxRange: 1}
	lance   = tactics.Weapon{Name: "鉄の槍", Type: tactics.Lance, Weight: 5, Strikes: 1, Might: 7, Hit: 80, MinRange: 1, MaxRange: 1}
	axe     = tactics.Weapon{Name: "鉄の斧", Type: tactics.Axe, Weight: 7, Strikes: 1, Might: 8, Hit: 70, MinRange: 1, MaxRange: 1}
	bow     = tactics.Weapon{Name: "鉄の弓", Type: tactics.Bow, Weight: 5, Strikes: 1, Might: 6, Hit: 85, MinRange: 2, MaxRange: 2}
	inspire = tactics.Skill{Name: "鼓舞", Effect: tactics.Inspire, Range: 2, Hit: 10, Avoid: 10}
	pavise  = tactics.Skill{Name: "大盾", Effect: tactics.Pavise, Chance: 30}
	wrath   = tactics.Skill{Name: "怒り", Effect: tactics.Wrath, Chance: 50}
	snipe   = tactics.Skill{Name: "狙撃", Effect: tactics.Snipe, Hit: 20}

	// クラスの基本値（設計書 10.1 節）。
	soldierBase  = tactics.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5, Lck: 2, Mag: 0, Res: 1}
	myrmidonBase = tactics.Stats{HP: 17, Str: 4, Skl: 8, Spd: 9, Def: 2, Mov: 5, Lck: 4, Mag: 1, Res: 1}
	fighterBase  = tactics.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5, Lck: 2, Mag: 0, Res: 0}
	archerBase   = tactics.Stats{HP: 17, Str: 5, Skl: 7, Spd: 6, Def: 3, Mov: 5, Lck: 4, Mag: 0, Res: 1}
)

// 本体のデータから ch01 の戦闘を作り、10.3 節・10.4 節の値（自軍の能力と出撃の枠は設計書（反復 2）10.3・10.4 節の値）と一致することを確かめる。
func TestNewBattleCh01(t *testing.T) {
	t.Run("TC-FR-UNT-01-01 自軍の能力・配置が 10.3 節・10.4 節のとおり", func(t *testing.T) {
		b, events := newCh01(t)
		if b.Turn != 1 || b.Phase != tactics.Player || b.Result != tactics.Ongoing {
			t.Errorf("ターン %d、番 %v、結果 %v", b.Turn, b.Phase, b.Result)
		}
		if len(events) != 1 || events[0].Kind != tactics.EventPhaseStart || events[0].Side != tactics.Player || events[0].Turn != 1 {
			t.Errorf("出来事 = %+v", events)
		}
		units := b.Units()
		if len(units) != 13 {
			t.Fatalf("ユニットの数 = %d（自軍 5、敵 8）", len(units))
		}
		want := []tactics.Unit{
			{Level: 1, Growth: tactics.Growth{HP: 55, Str: 45, Skl: 65, Spd: 70, Def: 20, Lck: 40, Mag: 20, Res: 20}, Name: "レオン", ClassName: "ロード", Side: tactics.Player, Stats: tactics.Stats{HP: 19, Str: 7, Skl: 11, Spd: 12, Def: 4, Mov: 5, Lck: 6, Mag: 2, Res: 2}, HP: 19, Pos: tactics.Pos{X: 1, Y: 14}, Weapon: sword, Items: []tactics.ItemStack{{Item: potion, Count: 2}}, Skill: inspire, Lord: true, Alive: true},
			{Level: 1, Growth: tactics.Growth{HP: 85, Str: 40, Skl: 20, Spd: 10, Def: 70, Lck: 20, Mag: 0, Res: 10}, Name: "ガルド", ClassName: "アーマー", Side: tactics.Player, Stats: tactics.Stats{HP: 24, Str: 6, Skl: 1, Spd: 2, Def: 11, Mov: 4, Lck: 2, Mag: 0, Res: 0}, HP: 24, Pos: tactics.Pos{X: 2, Y: 13}, Weapon: lance, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Skill: pavise, Alive: true},
			{Level: 1, Growth: tactics.Growth{HP: 70, Str: 75, Skl: 35, Spd: 30, Def: 15, Lck: 25, Mag: 0, Res: 10}, Name: "ブラン", ClassName: "ファイター", Side: tactics.Player, Stats: tactics.Stats{HP: 22, Str: 11, Skl: 5, Spd: 5, Def: 2, Mov: 5, Lck: 2, Mag: 0, Res: 0}, HP: 22, Pos: tactics.Pos{X: 3, Y: 14}, Weapon: axe, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Skill: wrath, Alive: true},
			{Level: 1, Growth: tactics.Growth{HP: 55, Str: 40, Skl: 60, Spd: 45, Def: 20, Lck: 35, Mag: 5, Res: 15}, Name: "ミラ", ClassName: "アーチャー", Side: tactics.Player, Stats: tactics.Stats{HP: 17, Str: 5, Skl: 9, Spd: 7, Def: 3, Mov: 5, Lck: 4, Mag: 0, Res: 1}, HP: 17, Pos: tactics.Pos{X: 1, Y: 15}, Weapon: bow, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Skill: snipe, Alive: true},
			{Level: 1, Growth: tactics.Growth{HP: 55, Str: 0, Skl: 40, Spd: 60, Def: 30, Lck: 45, Mag: 50, Res: 45}, Name: "リナ", ClassName: "僧侶", Side: tactics.Player, Stats: tactics.Stats{HP: 16, Str: 0, Skl: 2, Spd: 6, Def: 2, Mov: 5, Lck: 6, Mag: 0, Res: 5}, HP: 16, Pos: tactics.Pos{X: 2, Y: 15}, Weapon: tactics.Weapon{Name: "癒やしの杖", Type: tactics.Staff, Strikes: 1, Heal: 10, Uses: 15, MaxUses: 15, MinRange: 1, MaxRange: 1}, Items: []tactics.ItemStack{{Item: potion, Count: 1}}, Alive: true},
		}
		for i, w := range want {
			if !reflect.DeepEqual(units[i], w) {
				t.Errorf("units[%d] =\n%+v\nwant\n%+v", i, units[i], w)
			}
		}
	})

	t.Run("TC-FR-UNT-01-02 敵の能力・配置・動き方・ボスが 10.4 節のとおり", func(t *testing.T) {
		b, _ := newCh01(t)
		units := b.Units()
		boss := tactics.Stats{HP: 26, Str: 8, Skl: 7, Spd: 6, Def: 7, Mov: 5, Lck: 2, Res: 1}
		want := []struct {
			class  string
			stats  tactics.Stats
			weapon tactics.Weapon
			pos    tactics.Pos
			ai     tactics.AI
			boss   bool
		}{
			{"ソルジャー", soldierBase, lance, tactics.Pos{X: 6, Y: 10}, tactics.Charge, false},
			{"ソルジャー", soldierBase, lance, tactics.Pos{X: 7, Y: 4}, tactics.Hold, false},
			{"剣士", myrmidonBase, sword, tactics.Pos{X: 9, Y: 8}, tactics.Hold, false},
			{"剣士", myrmidonBase, sword, tactics.Pos{X: 11, Y: 3}, tactics.Hold, false},
			{"ファイター", fighterBase, axe, tactics.Pos{X: 10, Y: 11}, tactics.Charge, false},
			{"ファイター", fighterBase, axe, tactics.Pos{X: 13, Y: 8}, tactics.Hold, false},
			{"アーチャー", archerBase, bow, tactics.Pos{X: 12, Y: 7}, tactics.Hold, false},
			{"ソルジャー", boss, lance, tactics.Pos{X: 13, Y: 1}, tactics.Hold, true},
		}
		for i, w := range want {
			u := units[5+i]
			if u.Side != tactics.Enemy || u.ClassName != w.class || u.Stats != w.stats || u.HP != w.stats.HP ||
				!reflect.DeepEqual(u.Weapon, w.weapon) || u.Pos != w.pos || u.AI != w.ai || u.Boss != w.boss || u.Lord ||
				u.Skill != (tactics.Skill{}) || u.Items != nil || !u.Alive || u.Acted || u.Level != []int{1, 2, 3, 1, 2, 3, 1, 7}[i] || (w.boss && u.Name != "ハーゲン") {
				t.Errorf("units[%d] = %+v", 5+i, u)
			}
		}
	})

	t.Run("TC-FR-UNT-01-03 敵の名前に、名前のない同じクラスが 2 体以上なら増援も通して番号を付ける", func(t *testing.T) {
		b, _ := newCh01(t)
		var got []string
		for _, u := range b.Units()[5:] {
			got = append(got, u.Name)
		}
		want := []string{"ソルジャー 1", "ソルジャー 2", "剣士 1", "剣士 2", "ファイター 1", "ファイター 2", "アーチャー", "ハーゲン"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("敵の名前 = %q, want %q", got, want)
		}
		setup, err := core.NewSetup(loadBase(t), "ch01")
		if err != nil {
			t.Fatal(err)
		}
		got = nil
		for _, r := range setup.Reinforcements {
			got = append(got, r.Unit.Name)
		}
		want = []string{"ファイター 3", "ソルジャー 3", "剣士 3"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("増援の名前 = %q, want %q", got, want)
		}

		// 敵に 1 体、増援に 1 体だけのクラスは、通して数えるので番号が付く。
		// 敵だけに 1 体のクラスは番号が付かない。
		cat := smallCatalog(func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Reinforcements = []defs.Reinforcement{{Turn: 2, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 0, Y: 1}}}
			m.Enemies = append(m.Enemies, defs.EnemyPlacement{Level: 1, Class: "fighter", Weapon: "iron_axe", AI: "hold", X: 2, Y: 1})
		})
		setup, err = core.NewSetup(cat, "m")
		if err != nil {
			t.Fatal(err)
		}
		if got := []string{setup.Units[1].Name, setup.Units[2].Name, setup.Reinforcements[0].Unit.Name}; !reflect.DeepEqual(got, []string{"ソルジャー 1", "ファイター", "ソルジャー 2"}) {
			t.Errorf("名前 = %q", got)
		}
		named := smallCatalog(func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Enemies[0].Name = "隊長"
			m.Reinforcements = []defs.Reinforcement{
				{Turn: 2, EnemyPlacement: defs.EnemyPlacement{Level: 1, Name: "増援隊長", Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 0, Y: 1}},
				{Turn: 3, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "soldier", Weapon: "iron_lance", AI: "charge", X: 2, Y: 1}},
			}
		})
		ns, err := core.NewSetup(named, "m")
		if err != nil {
			t.Fatal(err)
		}
		if ns.Units[1].Name != "隊長" || ns.Reinforcements[0].Unit.Name != "増援隊長" || ns.Reinforcements[1].Unit.Name != "ソルジャー" {
			t.Fatalf("名前ありの敵と増援 = %+v", ns)
		}

	})

	t.Run("TC-FR-UNT-02-01 クラスによって武器の種類・移動力・能力が違う", func(t *testing.T) {
		b, _ := newCh01(t)
		classWeapon := map[string]tactics.WeaponType{
			"ロード": tactics.Sword, "アーマー": tactics.Lance, "ファイター": tactics.Axe,
			"僧侶": tactics.Staff, "アーチャー": tactics.Bow, "ソルジャー": tactics.Lance, "剣士": tactics.Sword,
		}
		for i, u := range b.Units() {
			if wt, ok := classWeapon[u.ClassName]; !ok || u.Weapon.Type != wt {
				t.Errorf("units[%d] %s（%s）の武器の種類 = %v", i, u.Name, u.ClassName, u.Weapon.Type)
			}
			wantMov := 5
			if u.ClassName == "アーマー" {
				wantMov = 4
			}
			if u.Stats.Mov != wantMov {
				t.Errorf("units[%d] %s の移動 = %d, want %d", i, u.Name, u.Stats.Mov, wantMov)
			}
		}
		// 同じクラスでボーナスのない敵は、同じ能力になる。クラスが違えば能力が違う。
		units := b.Units()
		if units[5].Stats != units[6].Stats || units[5].Stats == units[7].Stats {
			t.Errorf("ソルジャー %+v、%+v、剣士 %+v", units[5].Stats, units[6].Stats, units[7].Stats)
		}
	})

	t.Run("TC-FR-SYS-03-03 スキルの数値が skills.toml の値で、書き換えれば戦闘にも反映される", func(t *testing.T) {
		b, _ := newCh01(t)
		units := b.Units()
		for i, want := range []tactics.Skill{inspire, pavise, wrath, snipe} {
			if units[i].Skill != want {
				t.Errorf("%s のスキル = %+v, want %+v", units[i].Name, units[i].Skill, want)
			}
		}

		m := fstest.MapFS{}
		err := fs.WalkDir(data.Base(), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := fs.ReadFile(data.Base(), p)
			m[p] = &fstest.MapFile{Data: b}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		s := string(m["skills.toml"].Data)
		s = strings.Replace(s, "chance = 30", "chance = 50", 1)
		s = strings.Replace(s, "range = 2", "range = 3", 1)
		m["skills.toml"] = &fstest.MapFile{Data: []byte(s)}
		cat, err := defload.Load(m)
		if err != nil {
			t.Fatal(err)
		}
		b2, _, err := core.NewBattle(cat, "ch01", 1)
		if err != nil {
			t.Fatal(err)
		}
		u2 := b2.Units()
		if u2[0].Skill.Range != 3 || u2[1].Skill.Chance != 50 {
			t.Errorf("書き換えたスキル = %+v、%+v", u2[0].Skill, u2[1].Skill)
		}
	})

	t.Run("TC-FR-SYS-03-04 地形が 10.4 節の図の文字の地形になる", func(t *testing.T) {
		b, _ := newCh01(t)
		cat := loadBase(t)
		m, _ := cat.Map("ch01")
		bySymbol := map[rune]defs.TerrainDef{}
		for _, td := range cat.Terrains() {
			bySymbol[[]rune(td.Symbol)[0]] = td
		}
		if b.Width() != 16 || b.Height() != 16 {
			t.Fatalf("大きさ = %d × %d", b.Width(), b.Height())
		}
		for y, row := range m.Rows {
			for x, r := range []rune(row) {
				td := bySymbol[r]
				want := tactics.Terrain{Name: td.Name, Cost: td.Cost, Avoid: td.Avoid, Def: td.Def, HealPercent: td.HealPercent}
				if got := b.TerrainAt(tactics.Pos{X: x, Y: y}); !reflect.DeepEqual(got, want) {
					t.Errorf("(%d, %d) = %+v, want %+v", x, y, got, want)
				}
			}
		}
		fort := tactics.Terrain{Name: "砦", Cost: 1, Avoid: 20, Def: 2, HealPercent: 20}
		for _, p := range []tactics.Pos{{X: 13, Y: 1}, {X: 13, Y: 8}} {
			if got := b.TerrainAt(p); got != fort {
				t.Errorf("%v = %+v（砦のはず）", p, got)
			}
		}
		if got := b.TerrainAt(tactics.Pos{X: 12, Y: 3}); got != (tactics.Terrain{Name: "壁"}) {
			t.Errorf("(12, 3) = %+v（壁のはず）", got)
		}

		// 複数バイトの文字（森）の地形も、rune で数えて変える。
		small := smallCatalog(nil)
		setup, err := core.NewSetup(small, "m")
		if err != nil {
			t.Fatal(err)
		}
		if len(setup.Terrain) != 2 || len(setup.Terrain[1]) != 3 || setup.Terrain[1][1].Name != "森" || setup.Terrain[1][2].Name != "平地" {
			t.Errorf("地形 = %+v", setup.Terrain)
		}
	})

	t.Run("TC-FR-SYS-03-05 増援が reinforcements の順に、ターン・座標・動き方・能力を持つ", func(t *testing.T) {
		setup, err := core.NewSetup(loadBase(t), "ch01")
		if err != nil {
			t.Fatal(err)
		}
		want := []tactics.ReinforcementSetup{
			{Turn: 3, Unit: tactics.UnitSetup{Name: "ファイター 3", ClassName: "ファイター", Side: tactics.Enemy, Level: 1, Stats: fighterBase, Pos: tactics.Pos{X: 0, Y: 3}, Weapon: axe, AI: tactics.Charge}},
			{Turn: 3, Unit: tactics.UnitSetup{Name: "ソルジャー 3", ClassName: "ソルジャー", Side: tactics.Enemy, Level: 2, Stats: soldierBase, Pos: tactics.Pos{X: 0, Y: 4}, Weapon: lance, AI: tactics.Charge}},
			{Turn: 5, Unit: tactics.UnitSetup{Name: "剣士 3", ClassName: "剣士", Side: tactics.Enemy, Level: 3, Stats: myrmidonBase, Pos: tactics.Pos{X: 15, Y: 15}, Weapon: sword, AI: tactics.Charge}},
		}
		if !reflect.DeepEqual(setup.Reinforcements, want) {
			t.Errorf("増援 =\n%+v\nwant\n%+v", setup.Reinforcements, want)
		}
	})

	t.Run("TC-FR-ITM-03-01 自軍は回復薬を初めから持ち、敵の手持ちも定義のとおりになる", func(t *testing.T) {
		b, _ := newCh01(t)
		units := b.Units()
		for i, wantCount := range []int{2, 1, 1, 1, 1} {
			want := []tactics.ItemStack{{Item: potion, Count: wantCount}}
			if !reflect.DeepEqual(units[i].Items, want) {
				t.Errorf("%s の手持ち = %+v, want %+v", units[i].Name, units[i].Items, want)
			}
		}
		for _, u := range units[5:] {
			if len(u.Items) != 0 {
				t.Errorf("%s の手持ち = %+v（ch01 の敵は持たない）", u.Name, u.Items)
			}
		}
		cat := smallCatalog(func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Enemies[0].Items = []defs.ItemCount{{Item: "potion", Count: 3}}
		})
		setup, err := core.NewSetup(cat, "m")
		if err != nil {
			t.Fatal(err)
		}
		if want := []tactics.ItemStack{{Item: potion, Count: 3}}; !reflect.DeepEqual(setup.Units[1].Items, want) {
			t.Errorf("敵の手持ち = %+v, want %+v", setup.Units[1].Items, want)
		}
	})
}

// smallCatalog は、手で組み立てた小さな目録を返す。edit で、マップ "m" とキャラの列を書き換えられる。
//
// マップ "m" は 3 × 2（行 0 は平地、行 1 は「平地・森・平地」。森の文字は複数バイトの「森」）。
// 名簿はレオン（主人公）だけで、出撃の枠は (0, 0) の 1 つ。敵はボスのソルジャーが (2, 0)。
func smallCatalog(edit func(m *defs.MapDef, chars *[]defs.CharacterDef)) *defs.Catalog {
	return smallCatalogParty([]string{"leon"}, edit)
}

// smallCatalogParty は、名簿を party にした smallCatalog を返す。
func smallCatalogParty(party []string, edit func(m *defs.MapDef, chars *[]defs.CharacterDef)) *defs.Catalog {
	m := defs.MapDef{
		ID:      "m",
		Name:    "小さなマップ",
		Rows:    []string{"...", ".森."},
		Slots:   []defs.Slot{{X: 0, Y: 0}},
		Enemies: []defs.EnemyPlacement{{Level: 1, Class: "soldier", Weapon: "iron_lance", AI: "hold", Boss: true, X: 2, Y: 0}},
	}
	chars := []defs.CharacterDef{
		{ID: "leon", Name: "レオン", Class: "lord", Weapon: "iron_sword", Skill: "inspire", Lord: true},
	}
	if edit != nil {
		edit(&m, &chars)
	}
	return defs.NewCatalog(
		defs.GameDef{Party: party},
		[]defs.TerrainDef{
			{ID: "plain", Name: "平地", Symbol: ".", Cost: 1},
			{ID: "forest", Name: "森", Symbol: "森", Cost: 2, Avoid: 20, Def: 1},
		},
		[]defs.ClassDef{
			{ID: "lord", Name: "ロード", WeaponType: "sword", Base: defs.Stats{HP: 18, Str: 5, Skl: 7, Spd: 8, Def: 4, Mov: 5, Lck: 6, Mag: 2, Res: 2}},
			{ID: "soldier", Name: "ソルジャー", WeaponType: "lance", Base: defs.Stats{HP: 18, Str: 5, Skl: 4, Spd: 4, Def: 4, Mov: 5, Lck: 2, Mag: 0, Res: 1}},
			{ID: "fighter", Name: "ファイター", WeaponType: "axe", Base: defs.Stats{HP: 24, Str: 7, Skl: 4, Spd: 4, Def: 3, Mov: 5, Lck: 2, Mag: 0, Res: 0}},
			{ID: "armor_knight", Name: "アーマー", WeaponType: "lance", Tags: []string{"armor"}, Base: defs.Stats{HP: 22, Str: 6, Skl: 3, Spd: 2, Def: 9, Mov: 4, Lck: 2, Mag: 0, Res: 0}},
		},
		[]defs.WeaponDef{
			{ID: "iron_sword", Name: "鉄の剣", Type: "sword", Weight: 4, Strikes: 1, DamageType: "physical", Might: 5, Hit: 90, MinRange: 1, MaxRange: 1},
			{ID: "iron_lance", Name: "鉄の槍", Type: "lance", Weight: 5, Strikes: 1, DamageType: "physical", Might: 7, Hit: 80, MinRange: 1, MaxRange: 1},
			{ID: "iron_axe", Name: "鉄の斧", Type: "axe", Weight: 7, Strikes: 1, DamageType: "physical", Might: 8, Hit: 70, MinRange: 1, MaxRange: 1},
			{ID: "horse_lance", Name: "騎兵の槍", Type: "lance", Crit: 10, Weight: 6, Strikes: 2, DamageType: "physical", EffectiveTags: []string{"mounted"}, Might: 6, Hit: 75, MinRange: 1, MaxRange: 1},
			{ID: "gun", Name: "銃", Type: "gun", Might: 9, Hit: 90, MinRange: 1, MaxRange: 3},
		},
		[]defs.ItemDef{{ID: "potion", Name: "回復薬", Heal: 10}},
		[]defs.SkillDef{
			{ID: "inspire", Name: "鼓舞", Effect: "inspire", Range: 2, Hit: 10, Avoid: 10},
			{ID: "shield", Name: "盾", Effect: "shield"},
		},
		chars,
		[]defs.MapDef{m},
		[]defs.ChapterDef{{ID: "c1", Name: "章", Map: "m"}},
		nil,
	)
}

// 定義に誤りがあれば、戦闘を作らずに誤りを返すことを確かめる。
func TestNewBattleErrors(t *testing.T) {
	cat := loadBase(t)
	t.Run("TC-FR-UNT-01-04 存在しないマップの ID", func(t *testing.T) {
		for _, id := range []string{"ch99", ""} {
			b, events, err := core.NewBattle(cat, id, 1)
			if err == nil || !strings.Contains(err.Error(), "マップ \""+id+"\" がありません") || b != nil || events != nil {
				t.Errorf("NewBattle(%q) = %v, %v, %v", id, b, events, err)
			}
		}
	})

	type editFn = func(m *defs.MapDef, chars *[]defs.CharacterDef)
	tests := []struct {
		name string
		edit editFn
		want string
	}{
		{"名簿のキャラがない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].ID = "nobody" }, `slots[0]: party のキャラ "leon" がありません`},
		{"キャラのクラスがない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Class = "knight" }, `キャラ "leon" のクラス "knight" がありません`},
		{"キャラの武器がない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Weapon = "iron_rapier" }, `武器 "iron_rapier" がありません`},
		{"キャラのアイテムがない", func(_ *defs.MapDef, c *[]defs.CharacterDef) {
			(*c)[0].Items = []defs.ItemCount{{Item: "elixir", Count: 1}}
		}, `アイテム "elixir" がありません`},
		{"キャラのスキルがない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Skill = "luna" }, `スキル "luna" がありません`},
		{"スキルの効果の種類が使えない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Skill = "shield" }, `スキル "shield" の効果の種類 "shield" は使えません`},
		{"武器の種類が使えない", func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Weapon = "gun" }, `武器 "gun" の種類 "gun" は使えません`},
		{"敵のクラスがない", func(m *defs.MapDef, _ *[]defs.CharacterDef) { m.Enemies[0].Class = "dragon" }, `enemies[0]: クラス "dragon" がありません`},
		{"増援のクラスがない", func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Reinforcements = []defs.Reinforcement{{Turn: 2, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "dragon", Weapon: "iron_lance", AI: "charge", X: 0, Y: 1}}}
		}, `reinforcements[0]: クラス "dragon" がありません`},
		{"増援の武器がない", func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Reinforcements = []defs.Reinforcement{{Turn: 2, EnemyPlacement: defs.EnemyPlacement{Level: 1, Class: "soldier", Weapon: "iron_rapier", AI: "charge", X: 0, Y: 1}}}
		}, `reinforcements[0]: 武器 "iron_rapier" がありません`},
		{"敵のアイテムがない", func(m *defs.MapDef, _ *[]defs.CharacterDef) {
			m.Enemies[0].Items = []defs.ItemCount{{Item: "elixir", Count: 1}}
		}, `enemies[0]: アイテム "elixir" がありません`},
		{"敵の動き方が使えない", func(m *defs.MapDef, _ *[]defs.CharacterDef) { m.Enemies[0].AI = "flee" }, `enemies[0]: 動き方 "flee" は使えません`},
		{"地形の文字がない", func(m *defs.MapDef, _ *[]defs.CharacterDef) { m.Rows[1] = ".森X" }, `rows[1]: 3 文字目の地形の文字 "X" がありません`},
	}
	for _, tt := range tests {
		t.Run("TC-FR-UNT-01-05 "+tt.name, func(t *testing.T) {
			b, events, err := core.NewBattle(smallCatalog(tt.edit), "m", 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) || b != nil || events != nil {
				t.Errorf("NewBattle = %v, %v, %v\nwant 誤り %q", b, events, err, tt.want)
			}
		})
	}

	t.Run("TC-FR-UNT-01-06 tactics.New の誤りをそのまま返す", func(t *testing.T) {
		// 主人公のいない Setup になる（defload なら見つける誤り）。
		cat := smallCatalog(func(_ *defs.MapDef, c *[]defs.CharacterDef) { (*c)[0].Lord = false })
		b, events, err := core.NewBattle(cat, "m", 1)
		if !errors.Is(err, tactics.ErrInvalidSetup) || b != nil || events != nil {
			t.Errorf("NewBattle = %v, %v, %v", b, events, err)
		}
		// 正しい目録なら作れる。
		if _, _, err := core.NewBattle(smallCatalog(nil), "m", 1); err != nil {
			t.Errorf("正しい目録で NewBattle = %v", err)
		}
	})
}
