#!/usr/bin/env python3
"""#188 の JSON 計測結果から、移動経路の ZOC と増援の最終経路・全試行の表を作る。

使い方: summarize_path.py <新版の #188 計測 JSON(.gz)> <#174 の new.json.gz> <出力先ディレクトリ>
#174 の新版と、勝敗・ターン・巻き戻し・戦死者・最大生存敵が全試遊で一致することを検査する（計測がゲームを変えていない確認）。
"""
import argparse
import collections
import gzip
import json
from pathlib import Path


def read_report(path):
    opener = gzip.open if path.suffix == ".gz" else open
    with opener(path, "rt", encoding="utf-8") as stream:
        return json.load(stream)


def key(game):
    return game["Chapter"], game["Roster"], game["Player"], game["Seed"]


KINDS = ("PlayerMove", "PlayerRemove", "EnemyMove", "EnemyRemove")
FIELDS = ("Moves", "ZOCMoves", "PossibleMoves", "EntriesMin", "EntriesMax", "ExtraCost")


def total(games, path, kind):
    return {f: sum(g[path][kind][f] for g in games) for f in FIELDS}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("new188", type=Path)
    parser.add_argument("new174", type=Path)
    parser.add_argument("out", type=Path)
    args = parser.parse_args()
    cur, base = read_report(args.new188), read_report(args.new174)
    cur_games, base_games = {key(g): g for g in cur["Games"]}, {key(g): g for g in base["Games"]}
    if len(cur_games) != 920 or cur_games.keys() != base_games.keys():
        raise ValueError("#174 の新版と同じ 920 試遊でなければならない")
    for k, g in cur_games.items():
        for field in ("Result", "Turns", "Rewinds", "Fallen", "MaxLiveEnemies", "Deaths", "Deployment", "Route", "Order"):
            if g[field] != base_games[k][field]:
                raise ValueError(f"{k}: {field} が #174 の新版と違う")
        # 既存の着地点の数も同じはず。
        for field in ("PlayerZOCEndpointEntries", "EnemyZOCEndpointEntries", "ZOCExtraCost"):
            a = sum(t[field] for t in g["TurnDetails"])
            b = sum(t[field] for t in base_games[k]["TurnDetails"])
            if a != b:
                raise ValueError(f"{k}: {field} が #174 の新版と違う")
    groups = collections.defaultdict(list)
    for g in cur["Games"]:
        groups[key(g)[:3]].append(g)
    args.out.mkdir(parents=True, exist_ok=True)

    lines = [
        "# ZOC の移動経路の計測（最終経路、新版 920 試遊）",
        "",
        "各 EventMove の直前の盤面で最小費用の経路を求めて数えた。追加費用は最小費用の差（正確）、入る回数は最小費用の経路のうち最少〜最多（実際の経路は不明）。",
        "再移動は同じ番に攻撃・回復・待機のあとにした移動。自軍の打ち手は再移動を使わない。",
        "",
        "| 章 | 名簿 | 打ち手 | 敵 移動/再移動 | 敵 ZOC必須/可能 (通常) | 敵 入る回数 下限〜上限 (通常) | 敵 追加費用 (通常/再移動) | 自軍 移動 | 自軍 ZOC必須/可能 | 自軍 入る回数 下限〜上限 | 自軍 追加費用 |",
        "|---|---|---|---|---|---|---|---|---|---|---|",
    ]
    sums = {p: {k: collections.Counter() for k in KINDS} for p in ("PathFinal", "PathAllTrials")}
    for k in sorted(groups):
        games = groups[k]
        em, er = total(games, "PathFinal", "EnemyMove"), total(games, "PathFinal", "EnemyRemove")
        pm = total(games, "PathFinal", "PlayerMove")
        lines.append(
            f"| {k[0]} | {k[1]} | {k[2]} | {em['Moves']}/{er['Moves']} | {em['ZOCMoves']}/{em['PossibleMoves']} | {em['EntriesMin']}〜{em['EntriesMax']} | {em['ExtraCost']}/{er['ExtraCost']} "
            f"| {pm['Moves']} | {pm['ZOCMoves']}/{pm['PossibleMoves']} | {pm['EntriesMin']}〜{pm['EntriesMax']} | {pm['ExtraCost']} |"
        )
        for p in sums:
            for kind in KINDS:
                sums[p][kind].update(total(games, p, kind))
    lines += ["", "## 合計（最終経路 / 全試行）", "", "| 区分 | 移動 | ZOC必須 | ZOC可能 | 入る回数 下限 | 入る回数 上限 | 追加費用 |", "|---|---|---|---|---|---|---|"]
    names = {"PlayerMove": "自軍 通常", "PlayerRemove": "自軍 再移動", "EnemyMove": "敵 通常", "EnemyRemove": "敵 再移動"}
    for kind in KINDS:
        a, b = sums["PathFinal"][kind], sums["PathAllTrials"][kind]
        lines.append(f"| {names[kind]} | " + " | ".join(f"{a[f]} / {b[f]}" for f in FIELDS) + " |")
    (args.out / "zoc-path.md").write_text("\n".join(lines) + "\n", encoding="utf-8")

    lines = [
        "# 増援の最終経路と全試行の区別（新版 920 試遊）",
        "",
        "最終経路は巻き戻しで捨てた記録を除いたもの。全試行は巻き戻した試行を含む。巻き戻す前だけは、全試行の出現のうち最終経路に出なかった個体。",
        "旧基準の未出現は #174 の「全試行で一度も出なかった個体」。最終経路の未出現は、最終経路で出なかった予定の増援（巻き戻す前だけ出た個体を含む）。",
        "",
        "| 章 | 名簿 | 打ち手 | 巻き戻し回数 | 全試行で出た | 最終経路で出た | 巻き戻す前だけ | 旧基準の未出現 | 最終経路の未出現 | 巻き戻す前だけ出た試遊 |",
        "|---|---|---|---|---|---|---|---|---|---|",
    ]
    rew_games = 0
    for k in sorted(groups):
        games = groups[k]
        n = lambda f: sum(len(g[f] or []) for g in games)
        only = sum(bool(g["ReinforcementsRewoundOnly"]) for g in games)
        rew_games += only
        lines.append(f"| {k[0]} | {k[1]} | {k[2]} | {sum(g['Rewinds'] for g in games)} | {n('ReinforcementsAllTrials')} | {n('ReinforcementsFinal')} | {n('ReinforcementsRewoundOnly')} | {n('UnspawnedReinforcementIDs')} | {n('UnspawnedFinalPathIDs')} | {only} |")
    lines += ["", f"巻き戻す前だけ増援が出た試遊: {rew_games} / 920。"]
    (args.out / "reinforcement-final.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
