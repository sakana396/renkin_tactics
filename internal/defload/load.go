// Package defload は定義データ（TOML）を読み込み、意味を検証する（DES-PKG-03、ADR-0003）。
package defload

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// 定義データのファイル名（DES-DAT-02）。
const (
	FileGame       = "game.toml"
	FileTerrains   = "terrains.toml"
	FileClasses    = "classes.toml"
	FileWeapons    = "weapons.toml"
	FileItems      = "items.toml"
	FileSkills     = "skills.toml"
	FileCharacters = "characters.toml"
	FileMaps       = "maps.toml"
)

// マップの行の数と行の長さの上限（DES-DAT-03 規則 8。反復 1 はスクロールしない）。
const maxMapSize = 16

// Diagnostic は定義データの 1 つの誤り（DES-DAT-03）。
// 文法の誤りは Line（1 から数える）を、意味の誤りは Path（キーの道筋）を持つ。
type Diagnostic struct {
	File   string
	Line   int
	Path   string
	Reason string
}

// Error は「ファイル名: 位置: 理由」の形で誤りを表す（DES-DAT-03）。
func (d Diagnostic) Error() string {
	switch {
	case d.Line > 0:
		return fmt.Sprintf("%s: %d 行目: %s", d.File, d.Line, d.Reason)
	case d.Path != "":
		return fmt.Sprintf("%s: %s: %s", d.File, d.Path, d.Reason)
	default:
		return fmt.Sprintf("%s: %s", d.File, d.Reason)
	}
}

// Errors は定義データの誤りをすべて持つ（DES-DAT-03）。
type Errors []Diagnostic

// Error は誤りを 1 行に 1 つずつ並べる（DES-DAT-03）。
func (e Errors) Error() string {
	lines := make([]string, len(e))
	for i, d := range e {
		lines[i] = d.Error()
	}
	return strings.Join(lines, "\n")
}

// TOML の形（DES-DAT-02）。defs の型と分けて、TOML のキーの名前をここに閉じ込める。
type (
	gameFile struct {
		StartMap string `toml:"start_map"`
	}
	terrainsFile struct {
		Terrains []rawTerrain `toml:"terrains"`
	}
	classesFile struct {
		Classes []rawClass `toml:"classes"`
	}
	weaponsFile struct {
		Weapons []rawWeapon `toml:"weapons"`
	}
	itemsFile struct {
		Items []rawItem `toml:"items"`
	}
	skillsFile struct {
		Skills []rawSkill `toml:"skills"`
	}
	charactersFile struct {
		Characters []rawCharacter `toml:"characters"`
	}
	mapsFile struct {
		Maps []rawMap `toml:"maps"`
	}

	rawTerrain struct {
		ID          string `toml:"id"`
		Name        string `toml:"name"`
		Symbol      string `toml:"symbol"`
		Cost        int    `toml:"cost"`
		Avoid       int    `toml:"avoid"`
		Def         int    `toml:"def"`
		HealPercent int    `toml:"heal_percent"`
	}
	rawStats struct {
		HP  int `toml:"hp"`
		Str int `toml:"str"`
		Skl int `toml:"skl"`
		Spd int `toml:"spd"`
		Def int `toml:"def"`
		Mov int `toml:"mov"`
	}
	rawClass struct {
		ID         string   `toml:"id"`
		Name       string   `toml:"name"`
		WeaponType string   `toml:"weapon_type"`
		Base       rawStats `toml:"base"`
	}
	rawWeapon struct {
		ID       string `toml:"id"`
		Name     string `toml:"name"`
		Type     string `toml:"type"`
		Might    int    `toml:"might"`
		Hit      int    `toml:"hit"`
		MinRange int    `toml:"min_range"`
		MaxRange int    `toml:"max_range"`
	}
	rawItem struct {
		ID   string `toml:"id"`
		Name string `toml:"name"`
		Heal int    `toml:"heal"`
	}
	rawSkill struct {
		ID         string `toml:"id"`
		Name       string `toml:"name"`
		Effect     string `toml:"effect"`
		Range      int    `toml:"range"`
		Hit        int    `toml:"hit"`
		Avoid      int    `toml:"avoid"`
		Chance     int    `toml:"chance"`
		Multiplier int    `toml:"multiplier"`
	}
	rawItemCount struct {
		Item  string `toml:"item"`
		Count int    `toml:"count"`
	}
	rawCharacter struct {
		ID     string         `toml:"id"`
		Name   string         `toml:"name"`
		Class  string         `toml:"class"`
		Bonus  rawStats       `toml:"bonus"`
		Weapon string         `toml:"weapon"`
		Items  []rawItemCount `toml:"items"`
		Skill  string         `toml:"skill"`
		Lord   bool           `toml:"lord"`
	}
	rawAlly struct {
		Character string `toml:"character"`
		X         int    `toml:"x"`
		Y         int    `toml:"y"`
	}
	rawEnemy struct {
		Class  string         `toml:"class"`
		Weapon string         `toml:"weapon"`
		Bonus  rawStats       `toml:"bonus"`
		Items  []rawItemCount `toml:"items"`
		AI     string         `toml:"ai"`
		Boss   bool           `toml:"boss"`
		X      int            `toml:"x"`
		Y      int            `toml:"y"`
	}
	rawReinforcement struct {
		Turn   int            `toml:"turn"`
		Class  string         `toml:"class"`
		Weapon string         `toml:"weapon"`
		Bonus  rawStats       `toml:"bonus"`
		Items  []rawItemCount `toml:"items"`
		AI     string         `toml:"ai"`
		Boss   bool           `toml:"boss"`
		X      int            `toml:"x"`
		Y      int            `toml:"y"`
	}
	rawMap struct {
		ID             string             `toml:"id"`
		Name           string             `toml:"name"`
		Rows           []string           `toml:"rows"`
		Allies         []rawAlly          `toml:"allies"`
		Enemies        []rawEnemy         `toml:"enemies"`
		Reinforcements []rawReinforcement `toml:"reinforcements"`
	}
)

// Load は fsys の直下にある定義データのファイル（DES-DAT-02）をすべて読み、
// 意味を検証して目録を返す（DES-DAT-03、FR-SYS-03、FR-SYS-04）。
// 誤りがあれば、すべてを集めた Errors を返す。
// 文法の誤りや読めないファイルがあるときは、意味の検証はしない（中身が欠けたまま検証すると、見当違いの誤りが出るため）。
func Load(fsys fs.FS) (*defs.Catalog, error) {
	var errs Errors
	var (
		game       gameFile
		terrains   terrainsFile
		classes    classesFile
		weapons    weaponsFile
		items      itemsFile
		skills     skillsFile
		characters charactersFile
		maps       mapsFile
	)
	files := []struct {
		name string
		v    any
	}{
		{FileGame, &game},
		{FileTerrains, &terrains},
		{FileClasses, &classes},
		{FileWeapons, &weapons},
		{FileItems, &items},
		{FileSkills, &skills},
		{FileCharacters, &characters},
		{FileMaps, &maps},
	}
	var unknown Errors
	for _, f := range files {
		d, ok, u := decode(fsys, f.name, f.v)
		if !ok {
			errs = append(errs, d)
		}
		unknown = append(unknown, u...)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	// 知らないキーがあっても中身は読めているので、意味の検証も続ける（規則 13）。
	v := &validator{errs: unknown}
	cat := v.build(game, terrains.Terrains, classes.Classes, weapons.Weapons, items.Items, skills.Skills, characters.Characters, maps.Maps)
	if len(v.errs) > 0 {
		return nil, v.errs
	}
	return cat, nil
}

// decode は 1 つのファイルを読んで TOML を v に読み込む（DES-DAT-03 規則 1、13）。
// 読めなければ ok が false で、d に誤りが入る。読めたときは、知らないキーの誤りを unknown に返す。
func decode(fsys fs.FS, name string, v any) (d Diagnostic, ok bool, unknown []Diagnostic) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Diagnostic{File: name, Reason: fmt.Sprintf("ファイルを読めません（%v）", err)}, false, nil
	}
	md, err := toml.Decode(string(b), v)
	if err != nil {
		var pe toml.ParseError
		if errors.As(err, &pe) {
			msg := pe.Message
			if msg == "" {
				msg = pe.Error()
			}
			return Diagnostic{File: name, Line: pe.Position.Line, Reason: "TOML の文法の誤り（" + msg + "）"}, false, nil
		}
		// 値の型の誤り（文字列を数の欄に書いたなど）は ParseError にならず、行番号が文面にだけ入る。
		if m := typeErrPattern.FindStringSubmatch(err.Error()); m != nil {
			line, _ := strconv.Atoi(m[1])
			return Diagnostic{File: name, Line: line, Reason: "値の型の誤り（" + m[2] + "）"}, false, nil
		}
		return Diagnostic{File: name, Reason: "TOML を読めません（" + err.Error() + "）"}, false, nil
	}
	if len(md.Undecoded()) > 0 {
		unknown = unknownKeys(name, b, v, md.Undecoded())
	}
	return Diagnostic{}, true, unknown
}

// unknownKeys は定義にないキーを、配列の添字を含むキーの道筋で示す（DES-DAT-03 規則 13）。
// MetaData.Undecoded のキーには配列の添字がないため、TOML を汎用の形で読み直し、型と見比べて道筋を作る。
// 知らないキーかどうかは Undecoded を正とし、道筋から添字を除いたキーが Undecoded にあるものだけを示す（REV-006 N-01）。
func unknownKeys(name string, b []byte, v any, undecoded []toml.Key) []Diagnostic {
	known := make(map[string]bool, len(undecoded))
	for _, k := range undecoded {
		known[k.String()] = true
	}
	var out []Diagnostic
	var generic map[string]any
	if _, err := toml.Decode(string(b), &generic); err == nil {
		walkUnknown(generic, reflect.TypeOf(v), "", nil, func(path string, key toml.Key) {
			if known[key.String()] {
				out = append(out, Diagnostic{File: name, Path: path, Reason: "知らないキーです"})
			}
		})
	}
	if len(out) == 0 {
		// 道筋を作れなかったときは、Undecoded のキーをそのまま示す。
		for _, k := range undecoded {
			out = append(out, Diagnostic{File: name, Path: k.String(), Reason: "知らないキーです"})
		}
	}
	return out
}

// walkUnknown は値 x を型 t と見比べ、t にないキーの道筋を report に渡す。
// path は配列の添字を含む道筋、key は添字を除いたキー（MetaData.Undecoded と同じ形）。
// キーとタグの対応は toml と同じく、完全に一致するものを先に、なければ大文字・小文字だけが違うものを使う。
func walkUnknown(x any, t reflect.Type, path string, key toml.Key, report func(path string, key toml.Key)) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch val := x.(type) {
	case map[string]any:
		if t.Kind() != reflect.Struct {
			return
		}
		type field struct {
			tag string
			typ reflect.Type
		}
		var fields []field
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if tag := f.Tag.Get("toml"); tag != "" {
				fields = append(fields, field{tag, f.Type})
			}
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if path != "" {
				p = path + "." + k
			}
			kk := append(key[:len(key):len(key)], k)
			var ft reflect.Type
			for _, f := range fields {
				if f.tag == k {
					ft = f.typ
					break
				}
				if ft == nil && strings.EqualFold(f.tag, k) {
					ft = f.typ
				}
			}
			if ft == nil {
				report(p, kk)
				continue
			}
			walkUnknown(val[k], ft, p, kk, report)
		}
	case []map[string]any:
		if t.Kind() != reflect.Slice {
			return
		}
		for i, e := range val {
			walkUnknown(e, t.Elem(), fmt.Sprintf("%s[%d]", path, i), key, report)
		}
	case []any:
		if t.Kind() != reflect.Slice {
			return
		}
		for i, e := range val {
			walkUnknown(e, t.Elem(), fmt.Sprintf("%s[%d]", path, i), key, report)
		}
	}
}

// typeErrPattern は toml の値の型の誤りの文面から、行番号と理由を取り出す。
var typeErrPattern = regexp.MustCompile(`^toml: line (\d+) \(last key "[^"]*"\): (.*)$`)

// validator は意味の誤りを集める（DES-DAT-03）。
type validator struct {
	errs Errors
}

func (v *validator) add(file, path, format string, args ...any) {
	v.errs = append(v.errs, Diagnostic{File: file, Path: path, Reason: fmt.Sprintf(format, args...)})
}

// oneOf は値が許された値のどれかであることを確かめる（DES-DAT-03 規則 4）。
func (v *validator) oneOf(file, path, what, value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	v.add(file, path, "%s %q は使えません（%s のどれか）", what, value, strings.Join(allowed, "・"))
	return false
}

// atLeast は値が min 以上であることを確かめる（DES-DAT-03 規則 6）。
func (v *validator) atLeast(file, path string, value, min int) {
	if value < min {
		v.add(file, path, "%d 以上にしてください（%d）", min, value)
	}
}

// between は値が min 以上 max 以下であることを確かめる（DES-DAT-03 規則 6）。
func (v *validator) between(file, path string, value, min, max int) {
	if value < min || value > max {
		v.add(file, path, "%d 以上 %d 以下にしてください（%d）", min, max, value)
	}
}

// ids は ID が空でなく、重ならないことを確かめる（DES-DAT-03 規則 2）。
type ids struct {
	v    *validator
	file string
	seen map[string]string
}

func (v *validator) ids(file string) *ids {
	return &ids{v: v, file: file, seen: map[string]string{}}
}

func (s *ids) check(path, id string) {
	if id == "" {
		s.v.add(s.file, path+".id", "ID が空です")
		return
	}
	if first, ok := s.seen[id]; ok {
		s.v.add(s.file, path+".id", "ID %q が重なっています（%s と同じ）", id, first)
		return
	}
	s.seen[id] = path
}

func toStats(s rawStats) defs.Stats {
	return defs.Stats{HP: s.HP, Str: s.Str, Skl: s.Skl, Spd: s.Spd, Def: s.Def, Mov: s.Mov}
}

func toItems(raw []rawItemCount) []defs.ItemCount {
	if len(raw) == 0 {
		return nil
	}
	out := make([]defs.ItemCount, len(raw))
	for i, r := range raw {
		out[i] = defs.ItemCount{Item: r.Item, Count: r.Count}
	}
	return out
}

// stats は、クラスの能力にボーナスを足した値の範囲を確かめる（DES-DAT-03 規則 6）。
// HP は 1 以上、ほかは 0 以上。
func (v *validator) stats(file, path string, base, bonus defs.Stats) {
	type field struct {
		key         string
		base, bonus int
		min         int
	}
	fields := []field{
		{"hp", base.HP, bonus.HP, 1},
		{"str", base.Str, bonus.Str, 0},
		{"skl", base.Skl, bonus.Skl, 0},
		{"spd", base.Spd, bonus.Spd, 0},
		{"def", base.Def, bonus.Def, 0},
		{"mov", base.Mov, bonus.Mov, 0},
	}
	for _, f := range fields {
		if sum := f.base + f.bonus; sum < f.min {
			v.add(file, path+"."+f.key, "能力（クラス %d ＋ ボーナス %d ＝ %d）は %d 以上にしてください", f.base, f.bonus, sum, f.min)
		}
	}
}

// build は定義を検証し、目録を作る（DES-DAT-03 規則 2〜12）。
func (v *validator) build(game gameFile, rt []rawTerrain, rc []rawClass, rw []rawWeapon, ri []rawItem, rs []rawSkill, rch []rawCharacter, rm []rawMap) *defs.Catalog {
	weaponTypes := []string{defs.WeaponSword, defs.WeaponLance, defs.WeaponAxe, defs.WeaponBow}

	// 地形（規則 2、6、7）。
	terrains := make([]defs.TerrainDef, len(rt))
	bySymbol := map[string]defs.TerrainDef{}
	symbolAt := map[string]string{}
	tid := v.ids(FileTerrains)
	for i, r := range rt {
		p := fmt.Sprintf("terrains[%d]", i)
		tid.check(p, r.ID)
		v.atLeast(FileTerrains, p+".cost", r.Cost, 0)
		v.atLeast(FileTerrains, p+".avoid", r.Avoid, 0)
		v.atLeast(FileTerrains, p+".def", r.Def, 0)
		v.between(FileTerrains, p+".heal_percent", r.HealPercent, 0, 100)
		t := defs.TerrainDef{ID: r.ID, Name: r.Name, Symbol: r.Symbol, Cost: r.Cost, Avoid: r.Avoid, Def: r.Def, HealPercent: r.HealPercent}
		terrains[i] = t
		if n := utf8.RuneCountInString(r.Symbol); n != 1 {
			v.add(FileTerrains, p+".symbol", "地形の文字 %q は 1 文字にしてください（%d 文字）", r.Symbol, n)
			continue
		}
		if first, ok := symbolAt[r.Symbol]; ok {
			v.add(FileTerrains, p+".symbol", "地形の文字 %q が重なっています（%s と同じ）", r.Symbol, first)
			continue
		}
		symbolAt[r.Symbol] = p
		bySymbol[r.Symbol] = t
	}

	// クラス（規則 2、4。能力の範囲はキャラ・敵でボーナスを足したあとに確かめる）。
	classes := make([]defs.ClassDef, len(rc))
	classByID := map[string]defs.ClassDef{}
	cid := v.ids(FileClasses)
	for i, r := range rc {
		p := fmt.Sprintf("classes[%d]", i)
		cid.check(p, r.ID)
		v.oneOf(FileClasses, p+".weapon_type", "武器の種類", r.WeaponType, weaponTypes...)
		c := defs.ClassDef{ID: r.ID, Name: r.Name, WeaponType: r.WeaponType, Base: toStats(r.Base)}
		classes[i] = c
		if _, ok := classByID[r.ID]; !ok {
			classByID[r.ID] = c
		}
	}

	// 武器（規則 2、4、6）。
	weapons := make([]defs.WeaponDef, len(rw))
	weaponByID := map[string]defs.WeaponDef{}
	wid := v.ids(FileWeapons)
	for i, r := range rw {
		p := fmt.Sprintf("weapons[%d]", i)
		wid.check(p, r.ID)
		v.oneOf(FileWeapons, p+".type", "武器の種類", r.Type, weaponTypes...)
		v.atLeast(FileWeapons, p+".might", r.Might, 0)
		v.atLeast(FileWeapons, p+".hit", r.Hit, 0)
		v.atLeast(FileWeapons, p+".min_range", r.MinRange, 1)
		if r.MaxRange < r.MinRange {
			v.add(FileWeapons, p+".max_range", "最大の射程 %d は最小の射程 %d 以上にしてください", r.MaxRange, r.MinRange)
		}
		w := defs.WeaponDef{ID: r.ID, Name: r.Name, Type: r.Type, Might: r.Might, Hit: r.Hit, MinRange: r.MinRange, MaxRange: r.MaxRange}
		weapons[i] = w
		if _, ok := weaponByID[r.ID]; !ok {
			weaponByID[r.ID] = w
		}
	}

	// アイテム（規則 2、6）。
	items := make([]defs.ItemDef, len(ri))
	itemIDs := map[string]bool{}
	iid := v.ids(FileItems)
	for i, r := range ri {
		p := fmt.Sprintf("items[%d]", i)
		iid.check(p, r.ID)
		v.atLeast(FileItems, p+".heal", r.Heal, 1)
		items[i] = defs.ItemDef{ID: r.ID, Name: r.Name, Heal: r.Heal}
		itemIDs[r.ID] = true
	}

	// スキル（規則 2、4、6）。
	skills := make([]defs.SkillDef, len(rs))
	skillIDs := map[string]bool{}
	sid := v.ids(FileSkills)
	for i, r := range rs {
		p := fmt.Sprintf("skills[%d]", i)
		sid.check(p, r.ID)
		v.oneOf(FileSkills, p+".effect", "スキルの効果の種類", r.Effect, defs.EffectInspire, defs.EffectPavise, defs.EffectWrath, defs.EffectSnipe)
		v.atLeast(FileSkills, p+".range", r.Range, 0)
		v.atLeast(FileSkills, p+".hit", r.Hit, 0)
		v.atLeast(FileSkills, p+".avoid", r.Avoid, 0)
		v.between(FileSkills, p+".chance", r.Chance, 0, 100)
		if r.Effect == defs.EffectWrath {
			v.atLeast(FileSkills, p+".multiplier", r.Multiplier, 1)
		} else {
			v.atLeast(FileSkills, p+".multiplier", r.Multiplier, 0)
		}
		skills[i] = defs.SkillDef{ID: r.ID, Name: r.Name, Effect: r.Effect, Range: r.Range, Hit: r.Hit, Avoid: r.Avoid, Chance: r.Chance, Multiplier: r.Multiplier}
		skillIDs[r.ID] = true
	}

	// 手持ちのアイテム（規則 3、6）。
	checkItems := func(file, path string, list []rawItemCount) {
		for j, it := range list {
			q := fmt.Sprintf("%s.items[%d]", path, j)
			if !itemIDs[it.Item] {
				v.add(file, q+".item", "アイテム %q がありません", it.Item)
			}
			v.atLeast(file, q+".count", it.Count, 1)
		}
	}
	// クラスと武器の参照、武器の種類、能力（規則 3、5、6）。
	checkUnit := func(file, path, class, weapon string, bonus defs.Stats) {
		c, okC := classByID[class]
		if !okC {
			v.add(file, path+".class", "クラス %q がありません", class)
		} else {
			v.stats(file, path+".bonus", c.Base, bonus)
		}
		w, okW := weaponByID[weapon]
		if !okW {
			v.add(file, path+".weapon", "武器 %q がありません", weapon)
		}
		if okC && okW && c.WeaponType != w.Type {
			v.add(file, path+".weapon", "武器 %q の種類 %s が、クラス %q の武器の種類 %s と違います", weapon, w.Type, class, c.WeaponType)
		}
	}

	// キャラ（規則 2、3、5、6）。
	characters := make([]defs.CharacterDef, len(rch))
	charByID := map[string]defs.CharacterDef{}
	chid := v.ids(FileCharacters)
	for i, r := range rch {
		p := fmt.Sprintf("characters[%d]", i)
		chid.check(p, r.ID)
		ch := defs.CharacterDef{ID: r.ID, Name: r.Name, Class: r.Class, Bonus: toStats(r.Bonus), Weapon: r.Weapon, Items: toItems(r.Items), Skill: r.Skill, Lord: r.Lord}
		checkUnit(FileCharacters, p, r.Class, r.Weapon, ch.Bonus)
		checkItems(FileCharacters, p, r.Items)
		if r.Skill != "" && !skillIDs[r.Skill] {
			v.add(FileCharacters, p+".skill", "スキル %q がありません", r.Skill)
		}
		characters[i] = ch
		if _, ok := charByID[r.ID]; !ok {
			charByID[r.ID] = ch
		}
	}

	// マップ（規則 2、3、4、8〜12）。
	maps := make([]defs.MapDef, len(rm))
	mapIDs := map[string]bool{}
	mid := v.ids(FileMaps)
	for i, r := range rm {
		p := fmt.Sprintf("maps[%d]", i)
		mid.check(p, r.ID)
		mapIDs[r.ID] = true
		maps[i] = v.mapDef(p, r, bySymbol, charByID, checkUnit, checkItems)
	}

	// 始めるマップ（規則 3）。
	if !mapIDs[game.StartMap] {
		v.add(FileGame, "start_map", "マップ %q がありません", game.StartMap)
	}

	return defs.NewCatalog(defs.GameDef{StartMap: game.StartMap}, terrains, classes, weapons, items, skills, characters, maps)
}

// mapDef は 1 つのマップを検証し、定義に変える（DES-DAT-03 規則 3、4、8〜12）。
func (v *validator) mapDef(p string, r rawMap, bySymbol map[string]defs.TerrainDef, charByID map[string]defs.CharacterDef,
	checkUnit func(file, path, class, weapon string, bonus defs.Stats),
	checkItems func(file, path string, list []rawItemCount)) defs.MapDef {
	const f = FileMaps
	m := defs.MapDef{ID: r.ID, Name: r.Name, Rows: r.Rows}

	// 地形の行（規則 3、8）。
	if n := len(r.Rows); n < 1 || n > maxMapSize {
		v.add(f, p+".rows", "行の数 %d は 1 以上 %d 以下にしてください", n, maxMapSize)
	}
	grid := make([][]rune, len(r.Rows))
	for y, row := range r.Rows {
		q := fmt.Sprintf("%s.rows[%d]", p, y)
		grid[y] = []rune(row)
		n := len(grid[y])
		if n < 1 || n > maxMapSize {
			v.add(f, q, "行の長さ %d は 1 以上 %d 以下にしてください", n, maxMapSize)
		} else if y > 0 && n != len(grid[0]) {
			v.add(f, q, "行の長さ %d が、rows[0] の長さ %d と違います", n, len(grid[0]))
		}
		for x, ch := range grid[y] {
			if _, ok := bySymbol[string(ch)]; !ok {
				v.add(f, q, "%d 文字目の地形の文字 %q がありません", x+1, string(ch))
			}
		}
	}

	// 座標がマップの中にあり、通れる地形であることを確かめる（規則 9）。
	inMap := func(q string, x, y int) bool {
		if y < 0 || y >= len(grid) || x < 0 || x >= len(grid[y]) {
			v.add(f, q, "座標 (%d, %d) がマップの外です", x, y)
			return false
		}
		if t, ok := bySymbol[string(grid[y][x])]; ok && t.Cost == 0 {
			v.add(f, q, "座標 (%d, %d) の地形 %q は通れません", x, y, t.ID)
		}
		return true
	}
	// 初めの配置どうしが重ならないことを確かめる（規則 9）。
	type pos struct{ x, y int }
	occupied := map[pos]string{}
	place := func(q string, x, y int) {
		if !inMap(q, x, y) {
			return
		}
		at := pos{x, y}
		if first, ok := occupied[at]; ok {
			v.add(f, q, "座標 (%d, %d) が %s と重なっています", x, y, first)
			return
		}
		occupied[at] = q
	}

	// 自軍（規則 3、9、10）。
	lords := 0
	placed := map[string]string{}
	for j, a := range r.Allies {
		q := fmt.Sprintf("%s.allies[%d]", p, j)
		m.Allies = append(m.Allies, defs.AllyPlacement{Character: a.Character, X: a.X, Y: a.Y})
		place(q, a.X, a.Y)
		ch, ok := charByID[a.Character]
		if !ok {
			v.add(f, q+".character", "キャラ %q がありません", a.Character)
			continue
		}
		if first, dup := placed[a.Character]; dup {
			v.add(f, q+".character", "キャラ %q を 2 回置いています（%s と同じ）", a.Character, first)
			continue
		}
		placed[a.Character] = q
		if ch.Lord {
			lords++
		}
	}
	if lords != 1 {
		v.add(f, p+".allies", "lord = true のキャラを、ちょうど 1 人置いてください（%d 人）", lords)
	}

	// 敵（規則 3、4、5、6、9、11）。
	aiKinds := []string{defs.AICharge, defs.AIHold}
	bosses := 0
	for j, e := range r.Enemies {
		q := fmt.Sprintf("%s.enemies[%d]", p, j)
		ep := defs.EnemyPlacement{Class: e.Class, Weapon: e.Weapon, Bonus: toStats(e.Bonus), Items: toItems(e.Items), AI: e.AI, Boss: e.Boss, X: e.X, Y: e.Y}
		m.Enemies = append(m.Enemies, ep)
		checkUnit(f, q, e.Class, e.Weapon, ep.Bonus)
		checkItems(f, q, e.Items)
		v.oneOf(f, q+".ai", "敵の動き方", e.AI, aiKinds...)
		place(q, e.X, e.Y)
		if e.Boss {
			bosses++
		}
	}
	if bosses != 1 {
		v.add(f, p+".enemies", "boss = true の敵を、ちょうど 1 体置いてください（%d 体）", bosses)
	}

	// 増援（規則 3、4、5、6、9、11、12）。増援は初めの配置と重なってもよい（現れるときにふさがっていれば現れない）。
	for j, e := range r.Reinforcements {
		q := fmt.Sprintf("%s.reinforcements[%d]", p, j)
		ep := defs.EnemyPlacement{Class: e.Class, Weapon: e.Weapon, Bonus: toStats(e.Bonus), Items: toItems(e.Items), AI: e.AI, Boss: e.Boss, X: e.X, Y: e.Y}
		m.Reinforcements = append(m.Reinforcements, defs.Reinforcement{Turn: e.Turn, EnemyPlacement: ep})
		v.atLeast(f, q+".turn", e.Turn, 2)
		checkUnit(f, q, e.Class, e.Weapon, ep.Bonus)
		checkItems(f, q, e.Items)
		v.oneOf(f, q+".ai", "敵の動き方", e.AI, aiKinds...)
		inMap(q, e.X, e.Y)
		if e.Boss {
			v.add(f, q+".boss", "増援はボスになれません")
		}
	}
	return m
}
