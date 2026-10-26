"""DES-IT06 / DES-FML-01 の文書の検算。製品のコードは変えない。

実行方法（リポジトリの直下で）: python3 docs/03_design/scripts/verify_des06.py
Python 3.11 以上（tomllib を使う）。標準ライブラリだけを使う。
"""
from pathlib import Path
from fractions import Fraction as Q
import re
import tomllib

ROOT = Path(__file__).resolve().parents[3]
checks = 0

def eq(name, actual, expected):
    global checks
    assert actual == expected, f'{name}: {actual!r} != {expected!r}'
    checks += 1


def clamp(x, lo=0, hi=100):
    return min(hi, max(lo, x))


def AS(spd, weight, strength):
    return max(0, spd - max(0, weight-strength))


def values(strength, magic, skill, speed, weight, might, hit, crit,
           defender_speed, defender_strength, defender_weight,
           defense, resistance, luck, t=0, terrain_def=0, terrain_avoid=0,
           magical=False, effective=False, wrath=False):
    a = AS(speed, weight, strength)
    d = AS(defender_speed, defender_weight, defender_strength)
    attack = (magic if magical else strength) + might*(3 if effective else 1) + t
    damage = max(0, attack - (resistance if magical else defense) - terrain_def)
    h = clamp(2*skill + hit + 15*t - 2*d - terrain_avoid)
    c = clamp(skill//2 + crit + (50 if wrath else 0) - luck)
    return a, d, attack, damage, h, c, int(a-d >= 4)

# DES-IT06 B1〜B7（入力は文書の例そのもの）
eq('B1', values(7,2,11,12,4,5,90,0,4,5,5,4,1,2,-1,1,20), (12,4,11,6,69,3,1))
eq('B2', values(7,2,11,12,7,4,80,0,2,6,5,11,0,2,-1,effective=True), (12,2,18,7,83,3,1))
eq('B3', values(2,3,5,6,2,5,85,0,2,6,5,11,1,2,terrain_def=2,terrain_avoid=20,magical=True), (6,2,8,5,71,0,1))
eq('B4', values(11,0,5,5,7,8,70,0,2,6,5,11,0,2,1,wrath=True), (5,2,20,9,91,50,0))
eq('B4 hit', 10+20 < 2*91, True)
eq('B4 critical', 49 < 50, True)
eq('B4 critical boundary', 50 < 50, False)
eq('B4 pavise', 29 < 30, True)
eq('B4 damage', 9*3//2, 13)
eq('B4 noncritical pavise', 9//2, 4)
eq('B5', values(11,0,5,5,14,6,65,0,2,5,5,4,0,2,1), (2,2,18,14,86,0,0))
eq('B6', values(7,2,11,12,8,6,80,0,4,7,7,3,0,2,1,1,20,magical=True), (11,4,9,8,89,3,1))
eq('B6 snipe hit', 2*9+85+20,123)
eq('B7 hit low',clamp(2-200),0)
eq('B7 crit low',clamp(1//2-99),0)
eq('B7 hit high',clamp(2*99+100),100)
eq('B7 crit high',clamp(99//2+80+50-10),100)
eq('B7 AS low',AS(1,14,0),0)
eq('hit69 miss boundary',70+68 < 2*69,False)
eq('hit69 hit boundary',70+67 < 2*69,True)
eq('zero critical pavise',0*3//2,0)

# 攻撃順: 各機会を武器の撃数へ展開する。
# 反撃する側（D）は、武器の撃数によらず各機会 1 回（E-89。Q-DES06-02）。
def order(a_as,d_as,a_strikes,d_strikes,counter=True):
    turns=['A']+(['D'] if counter else [])
    if a_as-d_as>=4: turns+=['A']
    elif d_as-a_as>=4 and counter: turns+=['D']
    return [u for u in turns for _ in range(a_strikes if u=='A' else 1)]
eq('B1 order',order(12,4,1,1),['A','D','A'])
eq('B5 order',order(2,2,2,1),['A','A','D'])
eq('B3 order (distance 2, no counter)',order(6,2,1,1,counter=False),['A','A'])
eq('brave attacker followup',order(8,4,2,1),list('AADAA'))
eq('brave defender followup (counter 1)',order(4,8,1,2),list('ADD'))
eq('both brave (counter 1)',order(8,4,2,2),list('AADAA'))
eq('max strikes A/D',(order(8,4,2,2).count('A'),order(4,8,2,2).count('D')),(4,2))

# 回復 H1、経験 G1〜G5、移行能力例、再移動 C1。
eq('H1 rina',10+min(10+0,24-10),20)
eq('H1 hanna',10+min(10+3,24-10),23)
eq('H1 alchemist',10+min(min(10+3,8),24-10),18)
eq('H1 grown rina',10+min(10+4,24-10),24)
eq('hanna Lv7 expected mag',0+6*Q(50,100),3)
eq('v1 rina Lv9 mag',0+8*50//100,4)
kill=lambda own,enemy:clamp(30+(enemy-own)*3,10,50)
eq('G1',kill(17,8),10)
eq('G2',kill(6,10),42)
eq('G3',kill(1,20),50)
eq('G4',divmod(99+42,100),(1,41))
eq('G5 exp discarded at20', (95+10)//100,1)
eq('max exp sum',99+50,149)
eq('C1 remaining',7-(2+1),4)
eq('C1 valid move',2+1 <= 4,True)
eq('C1 invalid move',5 <= 4,False)
eq('v1 slots',1+4,5)
eq('v1 overflow',1+5-5,1)

# 現行 F1〜F12。新式ヘルパーに依存せず直接算術する。
eq('F1 attack',7+5-1,11)
eq('F1 damage',11-4-1,6)
eq('F1 H',2*11+90-15-(2*4+20),69)
eq('F1 followup',12-4>=4,True)
eq('F2 damage',5+7+1-4,9)
eq('F2 H',2*4+80+15-2*12,79)
eq('F3 miss',70+68<138,False)
eq('F3 hit',70+67<138,True)
eq('F4 H0',clamp(2-200),0)
eq('F4 H100',clamp(298),100)
eq('F4 minimum rolls miss',0+0<0,False)
eq('F4 maximum rolls hit',99+99<200,True)
eq('F5 still',18+85+20+10-(8+20+10),95)
eq('F5 moving',18+85+10-(8+20+10),75)
eq('F6 damage',9*3//2,13)
eq('F6 incorrect order differs',9//2*3,12)
eq('F7 floor',23//2,11)
eq('F7 active',11<=23//2,True)
eq('F7 inactive',12<=23//2,False)
eq('F8 recovered',min(10,19-12),7)
eq('F8 remaining uses',15-1,14)
eq('F9 fort heal',min(23*20//100,23-12),4)
eq('F9 HP',12+4,16)
eq('F10 exp',divmod(90+30,100),(1,20))
eq('F10 growth',[int(r<g) for r,g in zip([12,75,40,44,90],[55,40,60,45,20])],[1,0,1,1,0])
eq('F11 reaches20',19+(95+10)//100,20)
eq('F12 a kill',6*2>=12,True)
eq('F12 b kill',10*1>=12,False)
eq('F12 a gain',6*80*2,960)
eq('F12 b gain',10*90*1,900)
eq('F12 a risk',5*70,350)
eq('F12 c risk',4*60,240)

# ソースの鉄・能力・地形を読み、本文の移行前提を検証。
def read_toml(name):
    return tomllib.loads((ROOT/'data/base'/name).read_text())
classes={c['id']:c for c in read_toml('classes.toml')['classes']}
chars={c['id']:c for c in read_toml('characters.toml')['characters']}
weights={'iron_sword':4,'iron_lance':5,'iron_axe':7,'iron_bow':5,'heal_staff':0}
expected={'leon':(7,12),'gald':(6,2),'bran':(11,5),'mira':(5,7),'elsa':(7,7),'tio':(8,7),'rina':(0,6),'hanna':(0,7)}
for who,(strength,speed) in expected.items():
    ch=chars[who]; base=classes[ch['class']]['base']; bonus=ch.get('bonus',{})
    eq(f'{who} source str',base['str']+bonus.get('str',0),strength)
    eq(f'{who} source spd',base['spd']+bonus.get('spd',0),speed)
    eq(f'{who} iron/staff AS',AS(speed,weights[ch['weapon']],strength),speed)
for cls,weapon in [('myrmidon','iron_sword'),('soldier','iron_lance'),('fighter','iron_axe'),('archer','iron_bow')]:
    eq(f'{cls} iron supports', classes[cls]['base']['str']>=weights[weapon],True)
# ファイア（重さ 2）: 錬金術師（力 2）は遅くならず、魔道士（力 1）は 1 遅くなる（5.1・5.3 節）。
weights['fire']=2
eq('noel fire AS',AS(6,weights['fire'],2),6)
eq('mage fire AS',AS(5,weights['fire'],1),4)
for w in read_toml('weapons.toml')['weapons']:
    if w['id']=='heal_staff': eq('staff base',(w['heal'],w['uses']),(10,15))
    else:
        expect={'iron_sword':(5,90),'iron_lance':(7,80),'iron_axe':(8,70),'iron_bow':(6,85)}[w['id']]
        eq(w['id'],(w['might'],w['hit']),expect)

# Lvの表の範囲を現行データ全件から再集計。
expected_ranges={
'ch01':((0,8),(0,3)), 'ch02':((2,8),(1,2)), 'ch03':((3,10),(2,4)),
'ch04':((2,9),(1,3)), 'ch05':((3,10),(2,4)), 'ch06':((4,11),(2,4)),
'ch07':((5,12),(3,5)), 'ch08':((5,12),(3,5)), 'ch09':((6,14),(3,5))}
for m in read_toml('maps.toml')['maps']:
    enemies=m['enemies']+m.get('reinforcements',[])
    spans=tuple((min(e.get('bonus',{}).get(k,0) for e in enemies),max(e.get('bonus',{}).get(k,0) for e in enemies)) for k in ['hp','str'])
    eq(m['id']+' bonus range',spans,expected_ranges[m['id']])
    if m['id']=='ch09':
        eq('ch09 count',len(enemies),34)
        boss=next(e for e in enemies if e.get('boss'))
        eq('ch09 boss stats',[classes[boss['class']]['base'][k]+boss['bonus'][k] for k in ['hp','str','skl','spd','def']],[38,12,9,8,9])
    if m['id']=='ch06':
        boss=next(e for e in enemies if e.get('boss'))
        eq('ch06 boss hp/def',[classes[boss['class']]['base'][k]+boss['bonus'][k] for k in ['hp','def']],[33,13])
# L1、分数で期待値の誤差を避ける。
eq('L1 soldier',[Q(b)+5*Q(g,100) for b,g in zip([18,5,4,4,4],[60,40,35,30,30])],[Q(21),Q(7),Q('5.75'),Q('5.5'),Q('5.5')])
eq('L1 fighter',[Q(b)+12*Q(g,100) for b,g in zip([24,7,4,4,3],[70,75,35,30,15])],[Q('32.4'),Q(16),Q('8.2'),Q('7.6'),Q('4.8')])

# すべての兵種と仲間の既存能力・成長率が原本どおりか照合。
design_text=(ROOT/'docs/03_design/design-it06.md').read_text()
def table_row(key):
    line=next(x for x in design_text.splitlines() if x.startswith('| '+key+' |'))
    return [x.strip() for x in line.strip('|').split('|')]
for cid,c in classes.items():
    row=table_row(cid)
    eq(cid+' full base table',row[1],'/'.join(str(c['base'][k]) for k in ['hp','str','skl','spd','def','mov']))
for ch in chars.values():
    row=table_row(ch['name']); base=classes[ch['class']]['base']; bonus=ch.get('bonus',{})
    eq(ch['id']+' full stats table',row[2],'/'.join(str(base[k]+bonus.get(k,0)) for k in ['hp','str','skl','spd','def','mov']))
    eq(ch['id']+' full growth table',row[4],'/'.join(str(ch['growth'][k]) for k in ['hp','str','skl','spd','def']))
for ident in ['iron_sword','iron_lance','iron_axe','iron_bow','armor_killer','hammer','horse_killer','killing_sword','brave_axe','long_bow','thunder_sword','fire','heal_staff']:
    row=next([x.strip() for x in line.strip('|').split('|')] for line in design_text.splitlines() if line.startswith('| '+ident+'／'))
    might,hit,crit,weight=map(int,row[3:7]); strikes=int(row[9])
    eq(ident+' numeric bounds',might>=0 and hit>=0 and 0<=crit<=100 and weight>=0 and strikes in (1,2),True)
    if ident in weights: eq(ident+' weight',weight,weights[ident])
    if ident=='brave_axe': eq('brave data',(might,hit,weight,strikes),(6,65,14,2))
    if ident=='killing_sword': eq('killer crit',crit,30)

# 文書中の表・相対リンク・確認点の重複と本文参照。
for path in [ROOT/'docs/03_design/design-it06.md',ROOT/'docs/03_design/formulas.md',ROOT/'docs/README.md']:
    previous=None
    for lineno,line in enumerate(path.read_text().splitlines(),1):
        if line.startswith('|'):
            width=line.count('|')
            if previous is not None: eq(f'{path.name}:{lineno} table columns',width,previous)
            previous=width
        else: previous=None
    for target in re.findall(r'\]\(([^)]+)\)',path.read_text()):
        if '://' not in target and not target.startswith('#'):
            eq(f'{path.name} link {target}',(path.parent/target.split('#')[0]).exists(),True)
text=(ROOT/'docs/03_design/design-it06.md').read_text()
questions=re.findall(r'^\| (Q-DES06-\d+) \|',text,re.M)
eq('question IDs',questions,[f'Q-DES06-{i:02d}' for i in range(1,13)])
# 表の新しい行: ノエル、ch08 の報酬、5 枠の表、加入の順。
noel=table_row('ノエル（錬金術師）')
eq('noel stats = alchemist base (3.2 / 3.3)',noel[2],table_row('alchemist（新）')[1])
eq('noel extra = alchemist extra',noel[3],table_row('alchemist（新）')[2])
eq('noel fire weight in 5.1',next(x for x in design_text.splitlines() if x.startswith('| fire／')).split('|')[7].strip(),'2')
eq('noel 5.3 row',table_row('ノエル')[1:],['2','ファイア 2','0','6 → 6'])
eq('ch08 reward',table_row('ch08')[1:3],['ホースキラー（2 本目）','ガルド・エルザ'])
rewards=[table_row(f'ch0{i}')[1] for i in range(1,9)]
eq('reward count',len(rewards),8)
eq('fire not rewarded','ファイア' not in rewards,True)
for who in ['レオン','ガルド','エルザ','ブラン','ミラ','ティオ','リナ','ハンナ']:
    row=next([x.strip() for x in line.strip('|').split('|')] for line in design_text.splitlines() if line.startswith('| '+who+' |') and len(line.split('|'))==8)
    eq(who+' slots',int(row[3])+int(row[4]),5)
leon_candidates=['鉄の剣','アーマーキラー','キルソード','雷の剣']
eq('leon max weapons',int(next(line for line in design_text.splitlines() if line.startswith('| レオン | 剣 |')).split('|')[4]),len(leon_candidates))
eq('joins order (E-89)','`ch02.joins = ["noel", "elsa"]`' in design_text,True)
# 出撃枠は全章 +1（E-90）。13 章の表の「今の枠」「今の敵の数」を data から照合し、新しい枠 = 今 + 1。
maps={m['id']:m for m in read_toml('maps.toml')['maps']}
chapters_data=read_toml('chapters.toml')['chapters']
party=read_toml('game.toml')['party']
roster=list(party); tbd11_total=41
for ch in chapters_data:
    # data にはまだノエルがいないので、ch02 の joins は設計どおり ["noel", "elsa"] として数える。
    joins=['noel','elsa'] if ch['id']=='ch02' else ch.get('joins',[])
    roster+=[j for j in joins if j not in roster]
    row=next([x.strip() for x in line.strip('|').split('|')] for line in design_text.splitlines() if line.startswith('| '+ch['id']+' | ') and len(line.split('|'))==7)
    m=maps[ch['map']]; enemies=len(m['enemies'])+len(m.get('reinforcements',[]))
    eq(ch['id']+' slots now',int(row[1]),len(m['slots']))
    eq(ch['id']+' slots +1',int(row[2]),len(m['slots'])+1)
    eq(ch['id']+' enemies now',int(row[4]),enemies)
    eq(ch['id']+' roster size',int(row[3]),len(roster))
    if ch['id'] in ('ch01','ch02'): eq(ch['id']+' everyone deploys',len(roster)<=int(row[2]),True)
    eq(ch['id']+' total within TBD-11',int(row[2])+enemies<=tbd11_total,True)
eq('ch09 total 41',7+34,41)
# 窓の寸法・地図縮尺。
eq('layout width',512+320,832)
eq('layout height',512+64,576)
eq('info and minimap height',360+152,512)
eq('minimap32',128//32,4)
eq('minimap16',128//16,8)
eq('minimap x centered',(512+(320-128)//2, 512+(320-128)//2+127),(608,735))
eq('minimap y (title 16, bottom 8)',(360+16,360+16+127,511-(360+16+127)),(376,503,8))
eq('story paragraphs after adding',5+1,6)
print(f'成功: {checks} 項目。B1〜B7・H1・G1〜G5・L1・C1、F1〜F12、既存の仲間とノエルの武器の重さ、今の 9 章の bonus の範囲、報酬と 5 枠の表、ch02 の加入の順、全章の出撃枠 +1 と名簿・敵の数、勇者の斧の反撃 1 回、ミニマップの位置、表の列・リンクを検算した。')
