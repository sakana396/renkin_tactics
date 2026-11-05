#!/usr/bin/env python3
"""#174 の JSON 計測結果から、旧新版の比較表と seed ごとの CSV を作る。"""
import argparse
import collections
import csv
import gzip
import json
from pathlib import Path


def read_report(path):
    opener = gzip.open if path.suffix == ".gz" else open
    with opener(path, "rt", encoding="utf-8") as stream:
        return json.load(stream)


def key(game):
    return game["Chapter"], game["Roster"], game["Player"], game["Seed"]


def counts(game):
    turns = game.get("TurnDetails") or []
    missing = game.get("UnspawnedReinforcementDetails") or []
    return {
        "出現試行後の未出現": sum(bool(item["CauseUnknown"]) for item in missing),
        "自軍ZOC着地": sum(turn["PlayerZOCEndpointEntries"] for turn in turns),
        "敵軍ZOC着地": sum(turn["EnemyZOCEndpointEntries"] for turn in turns),
        "小隊長撃破": sum(len(turn.get("CommandersDefeated") or []) for turn in turns),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("old", type=Path)
    parser.add_argument("new", type=Path)
    parser.add_argument("out", type=Path)
    args = parser.parse_args()
    reports = [read_report(args.old), read_report(args.new)]
    # 同じ名簿・seed・進路・行動順で比較していることを検査する。
    indexed = [{key(g): g for g in r["Games"]} for r in reports]
    if len(indexed[0]) != 920 or indexed[0].keys() != indexed[1].keys():
        raise ValueError("旧新版は同じ 920 試遊でなければならない")
    for k, game in indexed[0].items():
        for field in ("Deployment", "Route", "Order"):
            if game[field] != indexed[1][k][field]:
                raise ValueError(f"{k}: 条件 {field} が違う")
    args.out.mkdir(parents=True, exist_ok=True)
    with (args.out / "seeds.csv").open("w", encoding="utf-8", newline="") as stream:
        writer = csv.writer(stream, lineterminator="\n")
        writer.writerow(["版", "コミット", "章", "名簿", "打ち手", "seed", "結果", "ターン", "最大生存敵", "巻き戻し", "出現試行後の未出現", "自軍ZOC着地", "敵軍ZOC着地", "小隊長撃破", "倒れた仲間", "死亡局面"])
        for version, report in zip(("旧", "新"), reports):
            for game in report["Games"]:
                writer.writerow([version, report["Commit"], *key(game), game["Result"], game["Turns"], game["MaxLiveEnemies"], game["Rewinds"], *counts(game).values(), "・".join(game.get("Fallen") or []), " / ".join(game.get("Deaths") or [])])
    groups = [collections.defaultdict(list), collections.defaultdict(list)]
    for group, report in zip(groups, reports):
        for game in report["Games"]:
            group[key(game)[:3]].append(game)
    lines = ["| 章 | 名簿 | 打ち手 | 勝ち数 旧→新 /20 | ターン範囲 旧→新 | 最大生存敵 旧→新 | 試行後未出現の体数 旧→新 | 新版ZOC着地 自軍/敵軍 |", "|---|---|---|---|---|---|---|---|"]
    for k in sorted(groups[0]):
        pair = [group[k] for group in groups]
        wins = [sum(g["Result"] == "Won" for g in games) for games in pair]
        ranges = [f'{min(g["Turns"] for g in games)}〜{max(g["Turns"] for g in games)}' for games in pair]
        peaks = [max(g["MaxLiveEnemies"] for g in games) for games in pair]
        missing = [sum(counts(g)["出現試行後の未出現"] for g in games) for games in pair]
        zoc = [sum(counts(g)[field] for g in pair[1]) for field in ("自軍ZOC着地", "敵軍ZOC着地")]
        lines.append(f'| {k[0]} | {k[1]} | {k[2]} | {wins[0]}→{wins[1]} | {ranges[0]}→{ranges[1]} | {peaks[0]}→{peaks[1]} | {missing[0]}→{missing[1]} | {zoc[0]}/{zoc[1]} |')
    (args.out / "comparison.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
