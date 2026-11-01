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
	FileChapters   = "chapters.toml"
	FileStory      = "story.toml"
)

// 手持ちの数の合計の上限（DES-DAT-03 規則 17、DES-CMP-04）。
const maxHand = 5

// キャラのレベルの上限（DES-DAT-03 規則 16、設計書 10.2 節）。
const maxLevel = 20

// 物語の 1 段落の長さの上限（rune で数える。DES-DAT-03 規則 18、設計書 10.7 節）。
const maxParagraph = 200

// マップの行の数と行の長さの上限（DES-DAT-03 規則 8、DES-MAP-05）。
const maxMapSize = 32

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
		Party []string       `toml:"party"`
		Stock []rawItemCount `toml:"stock"`
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
	chaptersFile struct {
		Chapters []rawChapter `toml:"chapters"`
	}
	storyFile struct {
		Stories []rawStory `toml:"stories"`
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
		Lck int `toml:"lck"`
		Mag int `toml:"mag"`
		Res int `toml:"res"`
		Mov int `toml:"mov"`
	}
	rawClass struct {
		ID         string   `toml:"id"`
		Name       string   `toml:"name"`
		WeaponType string   `toml:"weapon_type"`
		Tags       []string `toml:"tags"`
		Skills     []string `toml:"skills"`
		Base       rawStats `toml:"base"`
		// StaffHealCap は杖の回復量の上限。省略は 0（上限なし）（DES-IT06 7.1 節）。
		StaffHealCap int `toml:"staff_heal_cap"`
	}
	rawWeapon struct {
		ID            string   `toml:"id"`
		Name          string   `toml:"name"`
		Type          string   `toml:"type"`
		Crit          int      `toml:"crit"`
		Weight        int      `toml:"weight"`
		Strikes       *int     `toml:"strikes"`
		DamageType    string   `toml:"damage_type"`
		EffectiveTags []string `toml:"effective_tags"`
		Might         int      `toml:"might"`
		Hit           int      `toml:"hit"`
		MinRange      int      `toml:"min_range"`
		MaxRange      int      `toml:"max_range"`
		// Heal・Uses は、キーを書いたかどうかで規則 21 を確かめるので、ポインタにする。
		Heal *int `toml:"heal"`
		Uses *int `toml:"uses"`
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
		Item   string `toml:"item"`
		Weapon string `toml:"weapon"`
		Count  int    `toml:"count"`
	}
	rawGrowth struct {
		HP  int `toml:"hp"`
		Str int `toml:"str"`
		Skl int `toml:"skl"`
		Spd int `toml:"spd"`
		Def int `toml:"def"`
		Lck int `toml:"lck"`
		Mag int `toml:"mag"`
		Res int `toml:"res"`
	}
	rawCharacter struct {
		ID    string   `toml:"id"`
		Name  string   `toml:"name"`
		Class string   `toml:"class"`
		Bonus rawStats `toml:"bonus"`
		// Level は、省略（1）と 0（誤り）を見分けるので、ポインタにする（規則 16）。
		Level  *int      `toml:"level"`
		Growth rawGrowth `toml:"growth"`
		Weapon string    `toml:"weapon"`
		// Weapons は武器の列（DES-ITM-02）。古い weapon と両方は書けない。
		Weapons []string       `toml:"weapons"`
		Items   []rawItemCount `toml:"items"`
		Skill   string         `toml:"skill"`
		Lord    bool           `toml:"lord"`
	}
	rawSlot struct {
		X int `toml:"x"`
		Y int `toml:"y"`
	}
	rawEnemy struct {
		// Level は敵のレベル。省略を 1 と読むので、ポインタにする（DES-IT06 7.3 節）。
		Level *int `toml:"level"`
		// Name は、省略と空の文字列を見分けるので、ポインタにする（規則 19）。
		Name     *string        `toml:"name"`
		Class    string         `toml:"class"`
		Weapon   string         `toml:"weapon"`
		Weapons  []string       `toml:"weapons"`
		Bonus    rawStats       `toml:"bonus"`
		Items    []rawItemCount `toml:"items"`
		Approach string         `toml:"approach"`
		AI       string         `toml:"ai"`
		Boss     bool           `toml:"boss"`
		X        int            `toml:"x"`
		Y        int            `toml:"y"`
		// ID は、省略と空の文字列を見分けるので、ポインタにする（DES-MAP-07）。
		ID    *string `toml:"id"`
		Group string  `toml:"group"`
	}
	rawReinforcement struct {
		Level    *int           `toml:"level"`
		Turn     int            `toml:"turn"`
		Name     *string        `toml:"name"`
		Class    string         `toml:"class"`
		Weapon   string         `toml:"weapon"`
		Weapons  []string       `toml:"weapons"`
		Bonus    rawStats       `toml:"bonus"`
		Items    []rawItemCount `toml:"items"`
		Approach string         `toml:"approach"`
		AI       string         `toml:"ai"`
		Boss     bool           `toml:"boss"`
		X        int            `toml:"x"`
		Y        int            `toml:"y"`
		ID       *string        `toml:"id"`
		Group    string         `toml:"group"`
	}
	// rawArea は出来事の範囲（DES-MAP-07）。両端を含む。
	rawArea struct {
		MinX int `toml:"min_x"`
		MinY int `toml:"min_y"`
		MaxX int `toml:"max_x"`
		MaxY int `toml:"max_y"`
	}
	// rawEvent は出来事 1 件（DES-MAP-07）。Area は、書いたかを見分けるのでポインタにする。
	rawEvent struct {
		ID      string     `toml:"id"`
		Trigger string     `toml:"trigger"`
		Unit    string     `toml:"unit"`
		Area    *rawArea   `toml:"area"`
		Effect  string     `toml:"effect"`
		Enemies []rawEnemy `toml:"enemies"`
		Group   string     `toml:"group"`
	}
	rawMap struct {
		ID             string             `toml:"id"`
		Name           string             `toml:"name"`
		Rows           []string           `toml:"rows"`
		Slots          []rawSlot          `toml:"slots"`
		Enemies        []rawEnemy         `toml:"enemies"`
		Reinforcements []rawReinforcement `toml:"reinforcements"`
		Events         []rawEvent         `toml:"events"`
	}
	rawChapter struct {
		ID     string         `toml:"id"`
		Name   string         `toml:"name"`
		Map    string         `toml:"map"`
		Before string         `toml:"before"`
		After  string         `toml:"after"`
		Joins  []string       `toml:"joins"`
		Reward []rawItemCount `toml:"reward"`
	}
	rawStory struct {
		ID         string   `toml:"id"`
		Paragraphs []string `toml:"paragraphs"`
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
		chapters   chaptersFile
		story      storyFile
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
		{FileChapters, &chapters},
		{FileStory, &story},
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
	cat := v.build(rawData{
		game:       game,
		terrains:   terrains.Terrains,
		classes:    classes.Classes,
		weapons:    weapons.Weapons,
		items:      items.Items,
		skills:     skills.Skills,
		characters: characters.Characters,
		maps:       maps.Maps,
		chapters:   chapters.Chapters,
		stories:    story.Stories,
	})
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

// classSkills は兵種のスキルの参照と重なりを確かめる（DES-IT06 8.1 節、DES-DAT-04）。
// スキルの定義を読んだあとに呼ぶ。参照のない ID と、同じ ID の重なりを拒む。
func (v *validator) classSkills(ref refs, classes []rawClass) {
	for i, r := range classes {
		seen := map[string]bool{}
		for j, id := range r.Skills {
			p := fmt.Sprintf("classes[%d].skills[%d]", i, j)
			if !ref.skillIDs[id] {
				v.add(FileClasses, p, "スキル %q がありません", id)
				continue
			}
			if seen[id] {
				v.add(FileClasses, p, "兵種のスキル %q が重なっています", id)
				continue
			}
			seen[id] = true
		}
	}
}

// tags は兵種の分類・武器の特効の分類が armor・mounted のどれかで、重ならないことを確かめる（DES-IT06 5.2 節、DES-DAT-04）。
func (v *validator) tags(file, path, what string, tags []string) {
	seen := map[string]bool{}
	for j, tag := range tags {
		q := fmt.Sprintf("%s[%d]", path, j)
		v.oneOf(file, q, what, tag, "armor", "mounted")
		if seen[tag] {
			v.add(file, q, "%s %q が重なっています", what, tag)
		}
		seen[tag] = true
	}
}

func toStats(s rawStats) defs.Stats {
	return defs.Stats{HP: s.HP, Str: s.Str, Skl: s.Skl, Spd: s.Spd, Def: s.Def, Mov: s.Mov, Lck: s.Lck, Mag: s.Mag, Res: s.Res}
}

func toItems(raw []rawItemCount) []defs.ItemCount {
	if len(raw) == 0 {
		return nil
	}
	out := make([]defs.ItemCount, len(raw))
	for i, r := range raw {
		out[i] = defs.ItemCount{Item: r.Item, Weapon: r.Weapon, Count: r.Count}
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
		{"lck", base.Lck, bonus.Lck, 0}, {"mag", base.Mag, bonus.Mag, 0}, {"res", base.Res, bonus.Res, 0},
	}
	for _, f := range fields {
		if sum := f.base + f.bonus; sum < f.min {
			v.add(file, path+"."+f.key, "能力（クラス %d ＋ ボーナス %d ＝ %d）は %d 以上にしてください", f.base, f.bonus, sum, f.min)
		}
	}
}

// rawData は読み込んだすべてのファイルの中身（DES-DAT-02）。
type rawData struct {
	game       gameFile
	terrains   []rawTerrain
	classes    []rawClass
	weapons    []rawWeapon
	items      []rawItem
	skills     []rawSkill
	characters []rawCharacter
	maps       []rawMap
	chapters   []rawChapter
	stories    []rawStory
}

// refs は、ほかの定義から参照を引くための表（DES-DAT-03 規則 3）。
type refs struct {
	bySymbol   map[string]defs.TerrainDef
	classByID  map[string]defs.ClassDef
	weaponByID map[string]defs.WeaponDef
	itemIDs    map[string]bool
	skillIDs   map[string]bool
	charByID   map[string]defs.CharacterDef
}

// build は定義を検証し、目録を作る（DES-DAT-03 規則 2〜22。反復 2 で変更）。
func (v *validator) build(rd rawData) *defs.Catalog {
	weaponTypes := []string{defs.WeaponSword, defs.WeaponLance, defs.WeaponAxe, defs.WeaponBow, defs.WeaponStaff, defs.WeaponMagic}
	ref := refs{
		bySymbol:   map[string]defs.TerrainDef{},
		classByID:  map[string]defs.ClassDef{},
		weaponByID: map[string]defs.WeaponDef{},
		itemIDs:    map[string]bool{},
		skillIDs:   map[string]bool{},
		charByID:   map[string]defs.CharacterDef{},
	}

	// 地形（規則 2、6、7）。
	terrains := make([]defs.TerrainDef, len(rd.terrains))
	symbolAt := map[string]string{}
	tid := v.ids(FileTerrains)
	for i, r := range rd.terrains {
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
		ref.bySymbol[r.Symbol] = t
	}

	// クラス（規則 2、4。能力の範囲はキャラ・敵でボーナスを足したあとに確かめる）。
	classes := make([]defs.ClassDef, len(rd.classes))
	cid := v.ids(FileClasses)
	for i, r := range rd.classes {
		p := fmt.Sprintf("classes[%d]", i)
		cid.check(p, r.ID)
		v.oneOf(FileClasses, p+".weapon_type", "武器の種類", r.WeaponType, weaponTypes...)
		c := defs.ClassDef{ID: r.ID, Name: r.Name, WeaponType: r.WeaponType, StaffHealCap: r.StaffHealCap, Base: toStats(r.Base), Tags: append([]string(nil), r.Tags...), Skills: append([]string(nil), r.Skills...)}
		v.tags(FileClasses, p+".tags", "兵種の分類", c.Tags)
		v.atLeast(FileClasses, p+".staff_heal_cap", c.StaffHealCap, 0)
		classes[i] = c
		if _, ok := ref.classByID[r.ID]; !ok {
			ref.classByID[r.ID] = c
		}
	}

	// 武器（規則 2、4、6、21）。
	weapons := make([]defs.WeaponDef, len(rd.weapons))
	wid := v.ids(FileWeapons)
	for i, r := range rd.weapons {
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

		w.Crit, w.Weight = r.Crit, r.Weight
		w.Strikes = 1
		if r.Strikes != nil {
			w.Strikes = *r.Strikes
		}
		w.DamageType = r.DamageType
		if w.DamageType == "" {
			w.DamageType = "physical"
		}
		w.EffectiveTags = append([]string(nil), r.EffectiveTags...)
		v.between(FileWeapons, p+".crit", w.Crit, 0, 100)
		v.atLeast(FileWeapons, p+".weight", w.Weight, 0)
		v.between(FileWeapons, p+".strikes", w.Strikes, 1, 2)
		v.oneOf(FileWeapons, p+".damage_type", "攻撃の区分", w.DamageType, "physical", "magic")
		v.tags(FileWeapons, p+".effective_tags", "特効の分類", w.EffectiveTags)
		if w.Type == defs.WeaponMagic && w.DamageType != "magic" {
			v.add(FileWeapons, p+".damage_type", "種類 magic の武器は魔法の攻撃（magic）だけです")
		}
		if w.Type == defs.WeaponStaff && (w.Crit != 0 || w.Strikes != 1 || len(w.EffectiveTags) > 0) {
			v.add(FileWeapons, p, "杖には必殺・特効・2 回攻撃を付けられません")
		}
		v.staff(p, r, &w)
		weapons[i] = w
		if _, ok := ref.weaponByID[r.ID]; !ok {
			ref.weaponByID[r.ID] = w
		}
	}

	// アイテム（規則 2、6）。
	items := make([]defs.ItemDef, len(rd.items))
	iid := v.ids(FileItems)
	for i, r := range rd.items {
		p := fmt.Sprintf("items[%d]", i)
		iid.check(p, r.ID)
		v.atLeast(FileItems, p+".heal", r.Heal, 1)
		items[i] = defs.ItemDef{ID: r.ID, Name: r.Name, Heal: r.Heal}
		ref.itemIDs[r.ID] = true
	}

	// スキル（規則 2、4、6）。
	skills := make([]defs.SkillDef, len(rd.skills))
	sid := v.ids(FileSkills)
	for i, r := range rd.skills {
		p := fmt.Sprintf("skills[%d]", i)
		sid.check(p, r.ID)
		v.oneOf(FileSkills, p+".effect", "スキルの効果の種類", r.Effect, defs.EffectInspire, defs.EffectPavise, defs.EffectWrath, defs.EffectSnipe, defs.EffectCanto)
		v.atLeast(FileSkills, p+".range", r.Range, 0)
		v.atLeast(FileSkills, p+".hit", r.Hit, 0)
		v.atLeast(FileSkills, p+".avoid", r.Avoid, 0)
		v.between(FileSkills, p+".chance", r.Chance, 0, 100)
		if r.Effect == defs.EffectWrath {
			v.between(FileSkills, p+".multiplier", r.Multiplier, 0, 0)
		} else {
			v.atLeast(FileSkills, p+".multiplier", r.Multiplier, 0)
		}
		skills[i] = defs.SkillDef{ID: r.ID, Name: r.Name, Effect: r.Effect, Range: r.Range, Hit: r.Hit, Avoid: r.Avoid, Chance: r.Chance, Multiplier: r.Multiplier}
		ref.skillIDs[r.ID] = true
	}
	v.classSkills(ref, rd.classes)

	// キャラ（規則 2、3、5、6、16、17）。
	characters := make([]defs.CharacterDef, len(rd.characters))
	chid := v.ids(FileCharacters)
	for i, r := range rd.characters {
		p := fmt.Sprintf("characters[%d]", i)
		chid.check(p, r.ID)
		level := 1
		if r.Level != nil {
			level = *r.Level
			v.between(FileCharacters, p+".level", level, 1, maxLevel)
		}
		g := defs.Growth{HP: r.Growth.HP, Str: r.Growth.Str, Skl: r.Growth.Skl, Spd: r.Growth.Spd, Def: r.Growth.Def, Lck: r.Growth.Lck, Mag: r.Growth.Mag, Res: r.Growth.Res}
		v.growth(p+".growth", g)
		weapons := v.weaponList(FileCharacters, p, r.Weapon, r.Weapons)
		ch := defs.CharacterDef{ID: r.ID, Name: r.Name, Class: r.Class, Bonus: toStats(r.Bonus), Level: level, Growth: g, Weapons: weaponIDs(weapons), Items: toItems(r.Items), Skill: r.Skill, Lord: r.Lord}
		v.unit(ref, FileCharacters, p, r.Class, weapons, ch.Bonus)
		v.hand(ref, FileCharacters, p, r.Items, true, len(weapons))
		if r.Skill != "" && !ref.skillIDs[r.Skill] {
			v.add(FileCharacters, p+".skill", "スキル %q がありません", r.Skill)
		}
		characters[i] = ch
		if _, ok := ref.charByID[r.ID]; !ok {
			ref.charByID[r.ID] = ch
		}
	}

	// 名簿と在庫（規則 3、10、17）。
	game := defs.GameDef{Party: append([]string(nil), rd.game.Party...), Stock: toItems(rd.game.Stock)}
	inParty := v.party(ref, rd.game.Party)
	for i, r := range rd.characters {
		if r.Lord && !inParty[r.ID] {
			v.add(FileCharacters, fmt.Sprintf("characters[%d].lord", i), "lord = true のキャラ %q が party にいません", r.ID)
		}
	}
	v.rewardList(ref, FileGame, "stock", rd.game.Stock)

	// マップ（規則 2、3、4、8〜12、19、22）。
	maps := make([]defs.MapDef, len(rd.maps))
	mapByID := map[string]defs.MapDef{}
	mid := v.ids(FileMaps)
	for i, r := range rd.maps {
		p := fmt.Sprintf("maps[%d]", i)
		mid.check(p, r.ID)
		maps[i] = v.mapDef(ref, p, r)
		if _, ok := mapByID[r.ID]; !ok {
			mapByID[r.ID] = maps[i]
		}
	}

	// 物語（規則 18、20）。
	stories := make([]defs.StoryDef, len(rd.stories))
	storyIDs := map[string]bool{}
	stid := v.ids(FileStory)
	for i, r := range rd.stories {
		p := fmt.Sprintf("stories[%d]", i)
		stid.check(p, r.ID)
		if len(r.Paragraphs) == 0 {
			v.add(FileStory, p+".paragraphs", "段落を 1 つ以上書いてください")
		}
		for j, para := range r.Paragraphs {
			q := fmt.Sprintf("%s.paragraphs[%d]", p, j)
			if para == "" {
				v.add(FileStory, q, "段落が空です")
			} else if n := utf8.RuneCountInString(para); n > maxParagraph {
				v.add(FileStory, q, "段落の長さ %d 文字は %d 文字以下にしてください", n, maxParagraph)
			}
		}
		stories[i] = defs.StoryDef{ID: r.ID, Paragraphs: append([]string(nil), r.Paragraphs...)}
		storyIDs[r.ID] = true
	}

	// 章（規則 3、14、15、17、20）。
	if len(rd.chapters) == 0 {
		v.add(FileChapters, "chapters", "章を 1 つ以上書いてください")
	}
	chapters := make([]defs.ChapterDef, len(rd.chapters))
	joinedAt := map[string]string{}
	chapid := v.ids(FileChapters)
	for i, r := range rd.chapters {
		p := fmt.Sprintf("chapters[%d]", i)
		chapid.check(p, r.ID)
		if m, ok := mapByID[r.Map]; !ok {
			v.add(FileChapters, p+".map", "マップ %q がありません", r.Map)
		} else if len(m.Slots) == 0 {
			v.add(FileChapters, p+".map", "マップ %q に出撃の枠（slots）がありません", r.Map)
		}
		for _, s := range []struct{ key, id string }{{"before", r.Before}, {"after", r.After}} {
			if s.id != "" && !storyIDs[s.id] {
				v.add(FileChapters, p+"."+s.key, "物語 %q がありません", s.id)
			}
		}
		for j, id := range r.Joins {
			q := fmt.Sprintf("%s.joins[%d]", p, j)
			if _, ok := ref.charByID[id]; !ok {
				v.add(FileChapters, q, "キャラ %q がありません", id)
				continue
			}
			if inParty[id] {
				v.add(FileChapters, q, "キャラ %q は party にもいます", id)
				continue
			}
			if first, dup := joinedAt[id]; dup {
				v.add(FileChapters, q, "キャラ %q は %s でも加わります", id, first)
				continue
			}
			joinedAt[id] = q
		}
		v.rewardList(ref, FileChapters, p+".reward", r.Reward)
		chapters[i] = defs.ChapterDef{ID: r.ID, Name: r.Name, Map: r.Map, Before: r.Before, After: r.After, Joins: append([]string(nil), r.Joins...), Reward: toItems(r.Reward)}
	}

	return defs.NewCatalog(game, terrains, classes, weapons, items, skills, characters, maps, chapters, stories)
}

// staff は、杖の回復量と使用回数を確かめ、w に入れる（DES-DAT-03 規則 6、21）。
// 杖は heal・uses が 1 以上で、might・hit が 0。杖でない武器は heal・uses を書けない（値が 0 でも誤り）。
func (v *validator) staff(p string, r rawWeapon, w *defs.WeaponDef) {
	const f = FileWeapons
	keys := []struct {
		key string
		val *int
		dst *int
	}{{"heal", r.Heal, &w.Heal}, {"uses", r.Uses, &w.Uses}}
	if r.Type != defs.WeaponStaff {
		for _, k := range keys {
			if k.val != nil {
				v.add(f, p+"."+k.key, "杖でない武器は %s を持てません", k.key)
			}
		}
		return
	}
	for _, k := range keys {
		if k.val == nil {
			v.add(f, p+"."+k.key, "杖は %s を 1 以上にしてください（書かれていません）", k.key)
			continue
		}
		*k.dst = *k.val
		if *k.val < 1 {
			v.add(f, p+"."+k.key, "杖は %s を 1 以上にしてください（%d）", k.key, *k.val)
		}
	}
	for _, k := range []struct {
		key string
		val int
	}{{"might", r.Might}, {"hit", r.Hit}} {
		if k.val != 0 {
			v.add(f, p+"."+k.key, "杖は %s を 0 にしてください（%d）", k.key, k.val)
		}
	}
}

// growth は成長率の各値が 0 以上 100 以下であることを確かめる（DES-DAT-03 規則 16）。
func (v *validator) growth(path string, g defs.Growth) {
	for _, f := range []struct {
		key string
		val int
	}{{"hp", g.HP}, {"str", g.Str}, {"skl", g.Skl}, {"spd", g.Spd}, {"def", g.Def}, {"lck", g.Lck}, {"mag", g.Mag}, {"res", g.Res}} {
		v.between(FileCharacters, path+"."+f.key, f.val, 0, 100)
	}
}

// weaponRef は武器の ID と、誤りを示すときの位置（DES-ITM-02）。
type weaponRef struct {
	id, path string
}

// weaponList は、古い weapon と新しい weapons を武器の列に直す（DES-ITM-02、DES-IT06 6.2 節）。
// 両方を書いたら拒む。weapon だけなら 1 本の列にし、位置は今までどおり path.weapon にする。
// どちらもなければ、weapon の参照の誤りとして v.unit が知らせる（空の ID を 1 本だけ持つ）。
func (v *validator) weaponList(file, path, weapon string, weapons []string) []weaponRef {
	if weapon != "" && len(weapons) > 0 {
		v.add(file, path+".weapons", "weapon と weapons の両方は書けません")
		return nil
	}
	if len(weapons) == 0 {
		return []weaponRef{{id: weapon, path: path + ".weapon"}}
	}
	out := make([]weaponRef, len(weapons))
	for k, id := range weapons {
		out[k] = weaponRef{id: id, path: fmt.Sprintf("%s.weapons[%d]", path, k)}
	}
	return out
}

// weaponIDs は武器の列の ID を順に返す。
func weaponIDs(ws []weaponRef) []string {
	if len(ws) == 0 {
		return nil
	}
	out := make([]string, len(ws))
	for k, w := range ws {
		out[k] = w.id
	}
	return out
}

// unit は、クラスと武器の参照、武器の種類、能力を確かめる（DES-DAT-03 規則 3、5、6、DES-ITM-02）。
// 武器の列のすべてが、クラスの持てる種類であること。
func (v *validator) unit(ref refs, file, path, class string, weapons []weaponRef, bonus defs.Stats) {
	c, okC := ref.classByID[class]
	if !okC {
		v.add(file, path+".class", "クラス %q がありません", class)
	} else {
		v.stats(file, path+".bonus", c.Base, bonus)
	}
	for _, wr := range weapons {
		w, okW := ref.weaponByID[wr.id]
		if !okW {
			v.add(file, wr.path, "武器 %q がありません", wr.id)
		}
		if okC && okW && c.WeaponType != w.Type {
			v.add(file, wr.path, "武器 %q の種類 %s が、クラス %q の武器の種類 %s と違います", wr.id, w.Type, class, c.WeaponType)
		}
	}
}

// hand は手持ちのアイテムの参照と数を確かめる（DES-DAT-03 規則 3、6）。
// limit が真（キャラの手持ち）なら、数の合計が上限以下であることも確かめる（規則 17）。
// 道具の数が上限以下でも、武器の本数 weapons と合わせて 5 枠を超えれば拒む（DES-ITM-02。敵も同じ）。
func (v *validator) hand(ref refs, file, path string, list []rawItemCount, limit bool, weapons int) {
	total := 0
	for j, it := range list {
		q := fmt.Sprintf("%s.items[%d]", path, j)
		if it.Weapon != "" {
			v.add(file, q+".weapon", "道具の手持ちに武器は書けません")
		}
		if !ref.itemIDs[it.Item] {
			v.add(file, q+".item", "アイテム %q がありません", it.Item)
		}
		v.atLeast(file, q+".count", it.Count, 1)
		total += it.Count
	}
	switch {
	case limit && total > maxHand:
		v.add(file, path+".items", "手持ちの数の合計 %d は %d 以下にしてください", total, maxHand)
	case total+weapons > maxHand:
		v.add(file, path+".items", "手持ちの枠の合計 %d（武器 %d 本 ＋ 道具 %d 個）は %d 以下にしてください", total+weapons, weapons, total, maxHand)
	}
}

// itemList は在庫と報酬のアイテムの参照と数を確かめる（DES-DAT-03 規則 3、17）。同じアイテムを 2 回書かない。
func (v *validator) itemList(ref refs, file, path string, list []rawItemCount) {
	seen := map[string]string{}
	for j, it := range list {
		q := fmt.Sprintf("%s[%d]", path, j)
		if it.Weapon != "" {
			v.add(file, q+".weapon", "道具の在庫に武器は書けません")
		}
		if !ref.itemIDs[it.Item] {
			v.add(file, q+".item", "アイテム %q がありません", it.Item)
		} else if first, dup := seen[it.Item]; dup {
			v.add(file, q+".item", "アイテム %q を 2 回書いています（%s と同じ）", it.Item, first)
		} else {
			seen[it.Item] = q
		}
		v.atLeast(file, q+".count", it.Count, 1)
	}
}

// rewardList は報酬の道具または武器の参照と個数を確かめる（DES-ITM-02、DES-IT06 6.2 節）。
func (v *validator) rewardList(ref refs, file, path string, list []rawItemCount) {
	seen := map[string]string{}
	for j, it := range list {
		q := fmt.Sprintf("%s[%d]", path, j)
		switch {
		case it.Item != "" && it.Weapon != "":
			v.add(file, q, "item と weapon の両方は書けません")
		case it.Item == "" && it.Weapon == "":
			v.add(file, q, "item または weapon を書いてください")
		case it.Item != "":
			if !ref.itemIDs[it.Item] {
				v.add(file, q+".item", "アイテム %q がありません", it.Item)
			}
		case it.Weapon != "":
			if _, ok := ref.weaponByID[it.Weapon]; !ok {
				v.add(file, q+".weapon", "武器 %q がありません", it.Weapon)
			}
		}
		key := it.Item + "/" + it.Weapon
		if first, ok := seen[key]; ok && (it.Item != "" || it.Weapon != "") {
			kind, label, id := "item", "アイテム", it.Item
			if it.Weapon != "" {
				kind, label, id = "weapon", "武器", it.Weapon
			}
			v.add(file, q+"."+kind, "%s %q を 2 回書いています（%s と同じ）", label, id, first)
		}
		seen[key] = q
		v.atLeast(file, q+".count", it.Count, 1)
	}
}

// party は名簿を確かめ、名簿にいるキャラの ID の集合を返す（DES-DAT-03 規則 3、10）。
// 1 人以上で、同じキャラが 2 回なく、lord = true のキャラがちょうど 1 人いること。
func (v *validator) party(ref refs, party []string) map[string]bool {
	in := map[string]bool{}
	if len(party) == 0 {
		v.add(FileGame, "party", "party に 1 人以上入れてください")
		return in
	}
	at := map[string]string{}
	lords := 0
	for j, id := range party {
		q := fmt.Sprintf("party[%d]", j)
		ch, ok := ref.charByID[id]
		if !ok {
			v.add(FileGame, q, "キャラ %q がありません", id)
			continue
		}
		if first, dup := at[id]; dup {
			v.add(FileGame, q, "キャラ %q を 2 回入れています（%s と同じ）", id, first)
			continue
		}
		at[id] = q
		in[id] = true
		if ch.Lord {
			lords++
		}
	}
	if lords != 1 {
		v.add(FileGame, "party", "lord = true のキャラを、ちょうど 1 人入れてください（%d 人）", lords)
	}
	return in
}

// enemyName は敵の名前を確かめて返す（DES-DAT-03 規則 19）。書くなら空でない。
func (v *validator) enemyName(q string, name *string) string {
	if name == nil {
		return ""
	}
	if *name == "" {
		v.add(FileMaps, q+".name", "名前が空です（書かないか、空でない名前を書いてください）")
	}
	return *name
}

// noStaff は、敵の武器の列のどれも杖でないことを確かめる（DES-DAT-03 規則 22、DES-ITM-02）。
func (v *validator) noStaff(ref refs, weapons []weaponRef) {
	for _, wr := range weapons {
		if w, ok := ref.weaponByID[wr.id]; ok && w.Type == defs.WeaponStaff {
			v.add(FileMaps, wr.path, "敵は杖 %q を持てません", wr.id)
		}
	}
}

// mapDef は 1 つのマップを検証し、定義に変える（DES-DAT-03 規則 3、4、8〜12、19、22）。
func (v *validator) mapDef(ref refs, p string, r rawMap) defs.MapDef {
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
			if _, ok := ref.bySymbol[string(ch)]; !ok {
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
		if t, ok := ref.bySymbol[string(grid[y][x])]; ok && t.Cost == 0 {
			v.add(f, q, "座標 (%d, %d) の地形 %q は通れません", x, y, t.ID)
		}
		return true
	}
	// 出撃の枠と初めの敵の配置が、どれも重ならないことを確かめる（規則 9）。
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

	// 出撃の枠（規則 9）。枠が 1 つ以上あることは、章から確かめる（規則 14）。
	for j, s := range r.Slots {
		q := fmt.Sprintf("%s.slots[%d]", p, j)
		m.Slots = append(m.Slots, defs.Slot{X: s.X, Y: s.Y})
		place(q, s.X, s.Y)
	}

	// 敵（規則 3、4、5、6、9、11、19、22）。
	bosses := 0
	enemyIDs := v.enemyIDs()
	if n := len(r.Enemies); n > maxLiveEnemies {
		v.add(f, p+".enemies", "初めの敵が %d 体います（%d 体まで）", n, maxLiveEnemies)
	}
	for j, e := range r.Enemies {
		q := fmt.Sprintf("%s.enemies[%d]", p, j)
		m.Enemies = append(m.Enemies, v.enemy(ref, q, fmt.Sprintf("enemies[%d]", j), e, enemyIDs))
		place(q, e.X, e.Y)
		if e.Boss {
			bosses++
		}
	}
	if bosses != 1 {
		v.add(f, p+".enemies", "boss = true の敵を、ちょうど 1 体置いてください（%d 体）", bosses)
	}

	// 増援（規則 3、4、5、6、9、11、12、19、22）。増援は初めの配置と重なってもよい（出現時は距離 2 以内の立てる空きマスを距離・Y・X の順で選び、候補なしなら出さず延期しない（DES-MAP-03））。
	for j, e := range r.Reinforcements {
		q := fmt.Sprintf("%s.reinforcements[%d]", p, j)
		v.atLeast(f, q+".turn", e.Turn, 2)
		ep := v.enemy(ref, q, fmt.Sprintf("reinforcements[%d]", j), e.enemy(), enemyIDs)
		m.Reinforcements = append(m.Reinforcements, defs.Reinforcement{Turn: e.Turn, EnemyPlacement: ep})
		inMap(q, e.X, e.Y)
		if e.Boss {
			v.add(f, q+".boss", "増援はボスになれません")
		}
	}

	// 出来事（DES-MAP-07）。
	m.Events = v.mapEvents(ref, p, r.Events, len(grid), width(grid), inMap, enemyIDs)
	approaches := map[string]string{}
	checkApproach := func(q string, e defs.EnemyPlacement) {
		if e.Group == "" {
			return
		}
		approach := e.Approach
		if approach == "" {
			approach = defs.ApproachDirect
		}
		if first, ok := approaches[e.Group]; ok && first != approach {
			v.add(f, q+".approach", "一団 %q の寄り方がそろっていません", e.Group)
		}
		approaches[e.Group] = approach
	}
	for j, e := range m.Enemies {
		checkApproach(fmt.Sprintf("%s.enemies[%d]", p, j), e)
	}
	for j, e := range m.Reinforcements {
		checkApproach(fmt.Sprintf("%s.reinforcements[%d]", p, j), e.EnemyPlacement)
	}
	for j, event := range m.Events {
		for k, e := range event.Enemies {
			checkApproach(fmt.Sprintf("%s.events[%d].enemies[%d]", p, j, k), e)
		}
	}
	return m
}

// enemy は敵 1 体（初めの敵・ターンの増援・出来事の増援で共通）を定義に変え、
// 名前・ID・兵種と武器・杖・手持ち・動き方を確かめる（DES-DAT-03 規則 4〜6、11、19、22、DES-MAP-07）。
// 座標とボスの確かめは、配置の種類ごとに呼ぶ側で行う。
func (v *validator) enemy(ref refs, q, generated string, e rawEnemy, ids *enemyIDSet) defs.EnemyPlacement {
	const f = FileMaps
	weapons := v.weaponList(f, q, e.Weapon, e.Weapons)
	ep := defs.EnemyPlacement{Name: v.enemyName(q, e.Name), Class: e.Class, Weapons: weaponIDs(weapons), Bonus: toStats(e.Bonus), Items: toItems(e.Items), AI: e.AI, Approach: e.Approach, Boss: e.Boss, X: e.X, Y: e.Y, Group: e.Group}
	// level の省略は 1 と読む。0 と 21 以上は拒む（DES-IT06 7.3 節）。
	ep.Level = 1
	if e.Level != nil {
		ep.Level = *e.Level
	}
	v.between(FileMaps, q+".level", ep.Level, 1, maxLevel)
	ep.ID = ids.check(q, generated, e.ID, e.Group)
	v.unit(ref, f, q, e.Class, weapons, ep.Bonus)
	v.noStaff(ref, weapons)
	v.hand(ref, f, q, e.Items, false, len(weapons))
	v.oneOf(f, q+".ai", "敵の動き方", e.AI, defs.AICharge, defs.AIHold)
	if ep.Approach != "" {
		v.oneOf(f, q+".approach", "敵の寄り方", ep.Approach, defs.ApproachDirect, defs.ApproachGroup)
	}
	if ep.Approach == defs.ApproachGroup && ep.Group == "" {
		v.add(f, q+".group", "一団の寄り方には一団の名前が必要です")
	}
	return ep
}

// enemy は増援の、ターンを除いた敵の欄を返す。
func (r rawReinforcement) enemy() rawEnemy {
	return rawEnemy{Level: r.Level, Name: r.Name, Class: r.Class, Weapon: r.Weapon, Weapons: r.Weapons, Bonus: r.Bonus, Items: r.Items, AI: r.AI, Approach: r.Approach, Boss: r.Boss, X: r.X, Y: r.Y, ID: r.ID, Group: r.Group}
}

// maxLiveEnemies は一度に生きている敵の上限（DES-IT06 9.3 節、E-91）。tactics の maxLiveEnemies と同じ値。
// defload は tactics に依存しないので、ここにも置く。
const maxLiveEnemies = 39

// width は地形の行 0 の長さを返す。行がなければ 0。
func width(grid [][]rune) int {
	if len(grid) == 0 {
		return 0
	}
	return len(grid[0])
}

// enemyIDSet はマップの中の敵の ID を集める（DES-MAP-07）。explicit は id を書いた敵だけ。
type enemyIDSet struct {
	v        *validator
	seen     map[string]string
	explicit map[string]bool
	groups   map[string]bool
}

func (v *validator) enemyIDs() *enemyIDSet {
	return &enemyIDSet{v: v, seen: map[string]string{}, explicit: map[string]bool{}, groups: map[string]bool{}}
}

// check は敵 1 体の ID を確かめ、一団 group を集め、定義に書く ID を返す。
// 省いたら generated（配置の種類 ＋ 定義の添字）で重なりを確かめ、定義の ID は空のままにする
// （出来事から参照できるのは id を書いた敵だけなので、作った ID は戦闘で使わない）。
// 空の ID と、初めの敵・すべての増援を通した重なりを拒む。
func (s *enemyIDSet) check(q, generated string, id *string, group string) string {
	if group != "" {
		s.groups[group] = true
	}
	got := generated
	if id != nil {
		if *id == "" {
			s.v.add(FileMaps, q+".id", "ID が空です（書かないか、空でない ID を書いてください）")
			return ""
		}
		got = *id
		s.explicit[got] = true
	}
	if first, ok := s.seen[got]; ok {
		s.v.add(FileMaps, q+".id", "敵の ID %q が重なっています（%s と同じ）", got, first)
	} else {
		s.seen[got] = q
	}
	if id == nil {
		return ""
	}
	return got
}

// mapEvents は出来事の列を検証し、定義に変える（DES-MAP-07 の拒むもの）。
// 出来事から参照する敵は、id を書いた敵でなければならない。参照の確かめは、すべての敵を集めたあとに行う。
func (v *validator) mapEvents(ref refs, p string, raw []rawEvent, height, w int, inMap func(q string, x, y int) bool, enemyIDs *enemyIDSet) []defs.MapEvent {
	const f = FileMaps
	triggers := []string{defs.TriggerEnemyMoved, defs.TriggerEnemyDefeated, defs.TriggerEnemyAttacked, defs.TriggerPlayerEntered}
	effects := []string{defs.EventReinforce, defs.EventChargeGroup}
	eventIDs := v.ids(f)
	var out []defs.MapEvent
	// 出来事の増援の敵の ID と一団を、参照を確かめる前に集める。
	for i, e := range raw {
		q := fmt.Sprintf("%s.events[%d]", p, i)
		me := defs.MapEvent{ID: e.ID, Trigger: e.Trigger, Unit: e.Unit, Effect: e.Effect, Group: e.Group}
		for j, en := range e.Enemies {
			qe := fmt.Sprintf("%s.enemies[%d]", q, j)
			me.Enemies = append(me.Enemies, v.enemy(ref, qe, fmt.Sprintf("events[%d].enemies[%d]", i, j), en, enemyIDs))
			inMap(qe, en.X, en.Y)
			if en.Boss {
				v.add(f, qe+".boss", "増援はボスになれません")
			}
		}
		if e.Area != nil {
			me.Area = defs.Area{MinX: e.Area.MinX, MinY: e.Area.MinY, MaxX: e.Area.MaxX, MaxY: e.Area.MaxY}
		}
		out = append(out, me)
	}
	for i, e := range raw {
		q := fmt.Sprintf("%s.events[%d]", p, i)
		eventIDs.check(q, e.ID)
		if v.oneOf(f, q+".trigger", "きっかけ", e.Trigger, triggers...) {
			if e.Trigger == defs.TriggerPlayerEntered {
				if e.Unit != "" {
					v.add(f, q+".unit", "player_entered に unit は書けません")
				}
				if e.Area == nil {
					v.add(f, q+".area", "player_entered には area が要ります")
				} else if a := e.Area; a.MinX > a.MaxX || a.MinY > a.MaxY {
					v.add(f, q+".area", "範囲 (%d, %d)〜(%d, %d) の向きが逆です", a.MinX, a.MinY, a.MaxX, a.MaxY)
				} else if a.MinX < 0 || a.MinY < 0 || a.MaxX >= w || a.MaxY >= height {
					v.add(f, q+".area", "範囲 (%d, %d)〜(%d, %d) が盤の外です", a.MinX, a.MinY, a.MaxX, a.MaxY)
				}
			} else {
				if e.Area != nil {
					v.add(f, q+".area", "%s に area は書けません", e.Trigger)
				}
				if e.Unit == "" {
					v.add(f, q+".unit", "%s には unit が要ります", e.Trigger)
				} else if !enemyIDs.explicit[e.Unit] {
					v.add(f, q+".unit", "敵 %q がいません（出来事から参照する敵には id を書いてください）", e.Unit)
				}
			}
		}
		if v.oneOf(f, q+".effect", "効果", e.Effect, effects...) {
			if e.Effect == defs.EventReinforce {
				if len(e.Enemies) == 0 {
					v.add(f, q+".enemies", "reinforce には増援の敵が 1 体以上要ります")
				}
				if e.Group != "" {
					v.add(f, q+".group", "reinforce に group は書けません")
				}
			} else {
				if len(e.Enemies) != 0 {
					v.add(f, q+".enemies", "charge_group に増援の敵は書けません")
				}
				if e.Group == "" {
					v.add(f, q+".group", "charge_group には group が要ります")
				} else if !enemyIDs.groups[e.Group] {
					v.add(f, q+".group", "一団 %q の敵がいません", e.Group)
				}
			}
		}
	}
	return out
}
