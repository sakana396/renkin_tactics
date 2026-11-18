package core

import (
	"fmt"
	"math"
)

// Craft は材料を同時に消費して道具を一つ作る（DES-NG-19）。
func (c *Campaign) Craft(id string) error {
	recipe, ok := c.cat.Recipe(id)
	if !ok {
		return fmt.Errorf("core: レシピ %q がありません", id)
	}
	if _, ok := c.cat.Item(recipe.Item); !ok {
		return fmt.Errorf("core: 道具 %q がありません", recipe.Item)
	}
	for _, ingredient := range recipe.Ingredients {
		if c.Materials[ingredient.Material] < ingredient.Count {
			material, _ := c.cat.Material(ingredient.Material)
			return fmt.Errorf("%sが %d 足りません", material.Name, ingredient.Count-c.Materials[ingredient.Material])
		}
	}
	for _, item := range c.Stock {
		if item.Item == recipe.Item && item.Count >= math.MaxInt {
			return fmt.Errorf("core: 道具の在庫が上限です")
		}
	}
	for _, ingredient := range recipe.Ingredients {
		c.Materials[ingredient.Material] -= ingredient.Count
	}
	c.Stock = changeItem(c.Stock, recipe.Item, 1)
	c.sortStock()
	return nil
}
