package ui

import (
	"fmt"

	"github.com/sakana396/renkin_tactics/internal/core/tactics"
)

// availableItems は移動先から使える品目を返す（DES-NG-16）。
func (c *Controller) availableItems() []int {
	var out []int
	u := c.battle.Units()[c.unit]
	for i, stack := range u.Items {
		it := stack.Item
		if it.Effect != "" {
			if len(c.itemTargets(i)) > 0 {
				out = append(out, i)
			}
		} else if it.Bomb || u.HP < u.Stats.HP {
			out = append(out, i)
		}
	}
	return out
}

// itemCommand は指定した対象に道具を使う入力を作る（DES-NG-16）。
func (c *Controller) itemCommand(k int, p tactics.Pos) tactics.Command {
	return tactics.Command{Unit: c.unit, To: c.dest, Action: tactics.UseItem, Item: k, ItemTarget: &p}
}

// itemTargets は core の予想と同じ検証で使える対象を求める（DES-NG-16）。
func (c *Controller) itemTargets(k int) []int {
	var out []int
	for i, u := range c.battle.Units() {
		p := u.Pos
		if i == c.unit {
			p = c.dest
		}
		if _, err := c.battle.ItemForecast(c.itemCommand(k, p)); err == nil {
			out = append(out, i)
		}
	}
	return out
}

func (c *Controller) clickItemTarget(p tactics.Pos) {
	for _, i := range c.itemTargets(c.item) {
		if c.Units()[i].Pos != p {
			continue
		}
		if c.target == i {
			c.act(c.itemCommand(c.item, p))
			return
		}
		c.target, c.message = i, ""
		return
	}
	_, err := c.battle.ItemForecast(c.itemCommand(c.item, p))
	if err != nil {
		c.message = fmt.Sprintf("使えません: %v", err)
	}
}

// itemGuide は使用前後と期限を予想に表示する（DES-NG-16）。
func (c *Controller) itemGuide() []string {
	if c.target < 0 {
		return []string{"左クリック: 対象を選ぶ　右クリック・Esc: 戻る"}
	}
	p := c.Units()[c.target].Pos
	f, err := c.battle.ItemForecast(c.itemCommand(c.item, p))
	if err != nil {
		return []string{err.Error()}
	}
	line := fmt.Sprintf("%s　力 %d→%d　魔力 %d→%d", f.Name, f.Before.Str, f.After.Str, f.Before.Mag, f.After.Mag)
	if f.Effect == "snare" {
		line = fmt.Sprintf("%s　移動 %d→%d", f.Name, f.Before.Mov, f.After.Mov)
	}
	return []string{line + "　次の自軍開始まで", "もう一度クリックで使う　右クリック・Esc: 対象を選び直す"}
}
