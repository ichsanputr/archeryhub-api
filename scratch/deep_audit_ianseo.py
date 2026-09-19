import pymysql, json, urllib.request, re

conn = pymysql.connect(host='151.243.222.93', port=30036, user='ichsan', password='12345', database='archeris')
cur = conn.cursor(pymysql.cursors.DictCursor)

cur.execute("SELECT id, slug, name, source_url, data_json FROM tournament_externals ORDER BY id ASC")
externals = cur.fetchall()

print("=" * 100)
print("DEEP AUDIT OF ALL 16 IANSEO EXTERNAL TOURNAMENTS")
print("=" * 100)

for t in externals:
    t_id = t['id']
    slug = t['slug']
    name = t['name']
    src = t['source_url']
    dj = json.loads(t['data_json']) if t['data_json'] else {}
    
    cats = dj.get('categories', [])
    qual = dj.get('qualifications', {})
    team_qual = dj.get('team_qualifications', {})
    brackets = dj.get('brackets', {})
    standings = dj.get('final_standings', {})
    
    qual_cat_count = len(qual) if isinstance(qual, dict) else (len(qual) if isinstance(qual, list) else 0)
    bracket_cat_count = len(brackets) if isinstance(brackets, dict) else (len(brackets) if isinstance(brackets, list) else 0)
    
    # Check match counts & completeness in brackets
    incomplete_cats = []
    complete_cats = []
    
    if isinstance(brackets, dict):
        for cname, phases in brackets.items():
            tot_m = 0
            empty_m = 0
            if isinstance(phases, list):
                for p in phases:
                    for m in p.get('matches', []):
                        tot_m += 1
                        a1 = str(m.get('archer1') or m.get('score1') or m.get('name1') or '').strip()
                        a2 = str(m.get('archer2') or m.get('score2') or m.get('name2') or '').strip()
                        if not a1 and not a2:
                            empty_m += 1
            if tot_m == 0 or empty_m == tot_m:
                incomplete_cats.append(f"{cname} (0 valid matches)")
            else:
                complete_cats.append(f"{cname} ({tot_m} matches)")
    
    print(f"\n[{t_id}] {name} ({slug})")
    print(f"    Source URL        : {src}")
    print(f"    Categories Count  : Total {len(cats)} | Qual {qual_cat_count} | Brackets {bracket_cat_count} | Standings {len(standings)}")
    print(f"    Complete Bracket  : {len(complete_cats)} categories")
    if incomplete_cats:
        print(f"    Incomplete Bracket: {len(incomplete_cats)} -> {incomplete_cats}")
    else:
        print(f"    Incomplete Bracket: 0 (All brackets have valid matches)")

