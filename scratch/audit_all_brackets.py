import pymysql, json

conn = pymysql.connect(host='151.243.222.93', port=30036, user='ichsan', password='12345', database='archeris')
cur = conn.cursor(pymysql.cursors.DictCursor)

print("=" * 80)
print("AUDIT BRACKETS ON ALL EXTERNAL TOURNAMENTS")
print("=" * 80)

cur.execute("SELECT id, slug, name, data_json FROM tournament_externals ORDER BY id ASC")
externals = cur.fetchall()

for t in externals:
    t_id = t['id']
    slug = t['slug']
    name = t['name']
    dj = json.loads(t['data_json']) if t['data_json'] else {}
    
    # 1. Check data_json brackets
    dj_brackets = dj.get('brackets', {})
    dj_bracket_count = len(dj_brackets) if isinstance(dj_brackets, (dict, list)) else 0
    
    # Check valid matches in dj_brackets
    dj_valid_matches = 0
    if isinstance(dj_brackets, dict):
        for cat, phases in dj_brackets.items():
            if isinstance(phases, list):
                for p in phases:
                    for m in p.get('matches', []):
                        if m.get('archer1') or m.get('score1') or m.get('archer2') or m.get('score2'):
                            dj_valid_matches += 1
            elif isinstance(phases, dict):
                for p, m_list in phases.items():
                    if isinstance(m_list, list):
                        dj_valid_matches += len(m_list)
                    elif m_list:
                        dj_valid_matches += 1
                        
    # 2. Check relational matches
    cur.execute("SELECT COUNT(*) as total, COUNT(athlete1_name) as with_name FROM tournament_external_matches WHERE tournament_id = %s", (t_id,))
    m_stat = cur.fetchone()
    rel_total = m_stat['total']
    rel_with_name = m_stat['with_name']
    
    status = ""
    if dj_bracket_count > 0 or rel_with_name > 0:
        status = f"[READY] ({dj_bracket_count} categories, {dj_valid_matches} valid matches in JSON, {rel_with_name} rel matches)"
    elif rel_total > 0:
        status = f"[PLACEHOLDER ONLY] ({rel_total} placeholder rows in DB, 0 in JSON - no elimination held on source)"
    else:
        status = "[NO ELIMINATION] (Qualification-only tournament on Ianseo / Source)"
        
    print(f"ID {t_id:2d} | {slug[:45]:<45} | {status}")

print("\n" + "=" * 80)
print("AUDIT BRACKETS ON INTERNAL TOURNAMENTS")
print("=" * 80)

cur.execute("SELECT uuid, slug, name FROM tournaments ORDER BY created_at DESC")
internals = cur.fetchall()

for t in internals:
    u = t['uuid']
    slug = t['slug']
    name = t['name']
    
    cur.execute("SELECT COUNT(*) as cnt FROM elimination_matches WHERE tournament_id = %s OR tournament_id = %s", (u, slug))
    elim_cnt = cur.fetchone()['cnt']
    
    cur.execute("SELECT COUNT(*) as cnt FROM tournament_categories WHERE tournament_id = %s OR tournament_id = %s", (u, slug))
    cat_cnt = cur.fetchone()['cnt']
    
    print(f"UUID {u[:15]} | {slug[:35]:<35} | Categories: {cat_cnt} | Elimination Matches: {elim_cnt}")
