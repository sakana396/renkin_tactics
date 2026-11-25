#!/usr/bin/env python3
"""DES-NG08 案の平地・スキル補正なしの計算例を検算する。"""
def speed(spd, weight, strength):
    return max(0, spd - max(0, weight - strength))

def hit(skl, weapon_hit, triangle, defender_speed):
    return min(100, max(0, 2 * skl + weapon_hit + 15 * triangle - 2 * defender_speed))

assert speed(3, 1, 0) == 2
assert max(0, 0 + 3 - 1) == 2
assert hit(0, 75, 0, speed(6, 0, 1)) == 63
assert (16 - 1) // 2 == 7
assert max(0, 4 + 5 - 1) == 8
assert hit(5, 85, 0, speed(3, 5, 7)) == 89
assert max(0, 5 + 5 * 3 - 9) == 11
assert hit(5, 70, 0, speed(3, 5, 7)) == 74
assert max(10, min(50, 30 + (1 - 1) * 3)) == 30
print('DES-NG08 計算例: OK')
