package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
	"github.com/sakana396/renkin_tactics/internal/defs"
)

// セーブの誤りは errors.Is で識別する（DES-CMP-08、DES-SAV-03）。
var (
	ErrSaveVersion = errors.New("core: セーブの版に対応していません")
	ErrInvalidSave = errors.New("core: セーブの中身が正しくありません")
)

// saveData とその内側の型は、ゲームの型から独立した保存形式（DES-SAV-01）。
type saveData struct {
	Version int          `json:"version"`
	Rule    string       `json:"rule"`
	Chapter string       `json:"chapter"`
	Cleared bool         `json:"cleared"`
	Roster  []saveMember `json:"roster"`
	Stock   []saveItem   `json:"stock"`
}
type saveMember struct {
	Character string     `json:"character"`
	Level     int        `json:"level"`
	Exp       int        `json:"exp"`
	Stats     saveStats  `json:"stats"`
	Items     []saveItem `json:"items"`
	Alive     bool       `json:"alive"`
}
type saveStats struct {
	HP  int `json:"hp"`
	Str int `json:"str"`
	Skl int `json:"skl"`
	Spd int `json:"spd"`
	Def int `json:"def"`
	Mov int `json:"mov"`
}
type saveItem struct {
	Item  string `json:"item"`
	Count int    `json:"count"`
}

// saveItems は空でも [] になる、元の配列と独立した保存用の写しを作る（DES-SAV-01）。
func saveItems(items []defs.ItemCount) []saveItem {
	out := make([]saveItem, len(items))
	for i, v := range items {
		out[i] = saveItem{Item: v.Item, Count: v.Count}
	}
	return out
}

// Save は章の状態を空白 2 つの JSON に書く。状態を変更しない（DES-SAV-01）。
func (c *Campaign) Save(w io.Writer) error {
	d := saveData{Version: 1, Rule: "withdraw", Cleared: c.Cleared, Roster: make([]saveMember, len(c.Roster)), Stock: saveItems(c.Stock)}
	switch c.Rule {
	case Withdraw:
	case Permadeath:
		d.Rule = "permadeath"
	default:
		return fmt.Errorf("%w: rule: %d", ErrInvalidRule, c.Rule)
	}
	if !c.Cleared {
		chapters := c.cat.Chapters()
		if c.Chapter < 0 || c.Chapter >= len(chapters) {
			return invalidSave("chapter", "章の添字が範囲外です")
		}
		d.Chapter = chapters[c.Chapter].ID
	}
	for i, m := range c.Roster {
		d.Roster[i] = saveMember{Character: m.Character, Level: m.Level, Exp: m.Exp, Stats: saveStats(m.Stats), Items: saveItems(m.Items), Alive: m.Alive}
	}
	sort.Slice(d.Stock, func(i, j int) bool { return d.Stock[i].Item < d.Stock[j].Item })
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("core: セーブを JSON に変換できません: %w", err)
	}
	data = append(data, '\n')
	n, err := w.Write(data)
	if err != nil {
		return fmt.Errorf("core: セーブを書けません: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("core: セーブを書けません: %w", io.ErrShortWrite)
	}
	return nil
}

// invalidSave は位置と理由を持つ読み込みの誤りを作る（DES-SAV-03）。
func invalidSave(path, reason string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidSave, path, reason)
}

// LoadCampaign は版を先に読み、全規則を満たしたときだけキャンペーンを返す（DES-SAV-03）。
// cat は検証済みの目録。加入をやり直したり、既存のキャンペーンを書き換えたりしない。
func LoadCampaign(cat *defs.Catalog, r io.Reader) (*Campaign, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: json: 読み込みに失敗しました: %w", ErrInvalidSave, err)
	}
	var header struct {
		Version *int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, invalidSave("json.version", err.Error())
	}
	if header.Version == nil {
		return nil, invalidSave("version", "版がないか数ではありません")
	}
	if *header.Version != 1 {
		return nil, fmt.Errorf("%w: version: %d（対応する版は 1）", ErrSaveVersion, *header.Version)
	}
	var d saveData
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, invalidSave("json", err.Error())
	}
	c := &Campaign{cat: cat, Cleared: d.Cleared}
	switch d.Rule {
	case "withdraw":
		c.Rule = Withdraw
	case "permadeath":
		c.Rule = Permadeath
	default:
		return nil, invalidSave("rule", "withdraw・permadeath のどちらでもありません")
	}
	chapters := cat.Chapters()
	c.Chapter = len(chapters)
	if d.Cleared {
		if d.Chapter != "" {
			return nil, invalidSave("chapter", "クリア済みの章は空にしてください")
		}
	} else {
		for i, ch := range chapters {
			if ch.ID == d.Chapter {
				c.Chapter = i
				break
			}
		}
		if c.Chapter == len(chapters) {
			return nil, invalidSave("chapter", "目録に章がありません")
		}
	}
	if len(d.Roster) == 0 {
		return nil, invalidSave("roster", "名簿は 1 人以上必要です")
	}
	seen := map[string]bool{}
	lords := 0
	for i, m := range d.Roster {
		path := fmt.Sprintf("roster[%d]", i)
		ch, ok := cat.Character(m.Character)
		if !ok {
			return nil, invalidSave(path+".character", "目録にキャラがありません")
		}
		if seen[m.Character] {
			return nil, invalidSave(path+".character", "キャラが重複しています")
		}
		seen[m.Character] = true
		if ch.Lord {
			lords++
			if !m.Alive {
				return nil, invalidSave(path+".alive", "主人公が生きていません")
			}
		}
	}
	if lords != 1 {
		return nil, invalidSave("roster", "主人公はちょうど 1 人必要です")
	}
	// 名簿全体を規則 5 → 6 → 7 の順で検証する。
	for i, m := range d.Roster {
		path := fmt.Sprintf("roster[%d]", i)
		if m.Level < 1 || m.Level > 20 {
			return nil, invalidSave(path+".level", "レベルは 1〜20 です")
		}
		if m.Exp < 0 || m.Exp > 99 || (m.Level == 20 && m.Exp != 0) {
			return nil, invalidSave(path+".exp", "経験は 0〜99、レベル 20 なら 0 です")
		}
		for _, s := range []struct {
			name       string
			value, min int
		}{{"hp", m.Stats.HP, 1}, {"str", m.Stats.Str, 0}, {"skl", m.Stats.Skl, 0}, {"spd", m.Stats.Spd, 0}, {"def", m.Stats.Def, 0}, {"mov", m.Stats.Mov, 0}} {
			if s.value < s.min {
				return nil, invalidSave(path+".stats."+s.name, "能力が下限より小さい値です")
			}
		}
	}
	c.Roster = make([]Member, len(d.Roster))
	for i, m := range d.Roster {
		items, err := loadItems(cat, m.Items, fmt.Sprintf("roster[%d].items", i), true)
		if err != nil {
			return nil, err
		}
		c.Roster[i] = Member{Character: m.Character, Level: m.Level, Exp: m.Exp, Stats: tactics.Stats(m.Stats), Items: items, Alive: m.Alive}
	}
	c.Stock, err = loadItems(cat, d.Stock, "stock", false)
	if err != nil {
		return nil, err
	}
	c.sortStock()
	return c, nil
}

// loadItems は参照・個数・手持ち上限または在庫の重複を検証する（DES-SAV-03 規則 7）。
func loadItems(cat *defs.Catalog, items []saveItem, path string, hand bool) ([]defs.ItemCount, error) {
	out := make([]defs.ItemCount, 0, len(items))
	seen := map[string]bool{}
	remaining := 5
	for i, v := range items {
		pos := fmt.Sprintf("%s[%d]", path, i)
		if _, ok := cat.Item(v.Item); !ok {
			return nil, invalidSave(pos+".item", "目録にアイテムがありません")
		}
		if v.Count < 1 {
			return nil, invalidSave(pos+".count", "数は 1 以上です")
		}
		if hand {
			// 減算で調べ、非常に大きな個数でも加算の桁あふれを起こさない。
			if v.Count > remaining {
				return nil, invalidSave(path, "手持ちの合計は 5 以下です")
			}
			remaining -= v.Count
		} else if seen[v.Item] {
			return nil, invalidSave(pos+".item", "在庫のアイテムが重複しています")
		}
		seen[v.Item] = true
		out = append(out, defs.ItemCount{Item: v.Item, Count: v.Count})
	}
	return out, nil
}
