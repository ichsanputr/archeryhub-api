import urllib.request, re, json, pymysql

def parse_ianseo_bracket_page(url):
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    html = urllib.request.urlopen(req).read().decode('utf-8', errors='ignore')
    
    # Extract all table rows
    rows = re.findall(r'<tr[^>]*>(.*?)</tr>', html, re.S)
    
    # Find all cells with their column indices
    parsed_matches = []
    
    # Pattern to match archer cell blocks
    # Ianseo bracket pages typically have columns for 1/8, 1/4, 1/2, Finals
    # Let's extract archers and pairings
    archer_entries = []
    for r in rows:
        cols = re.findall(r'<td[^>]*>(.*?)</td>', r, re.S)
        clean_cols = [re.sub(r'<[^>]+>', '', c).strip() for c in cols]
        
        # Check if row has an archer definition: seed, name, club, score
        for i in range(len(clean_cols)):
            c = clean_cols[i]
            # If seed followed by name
            if re.match(r'^\d{1,2}$', c) and i + 1 < len(clean_cols) and len(clean_cols[i+1]) > 3:
                seed = c
                name = clean_cols[i+1]
                club = clean_cols[i+3] if i + 3 < len(clean_cols) and len(clean_cols[i+3]) > 1 else (clean_cols[i+2] if i+2 < len(clean_cols) else '')
                score = ''
                # Look for score
                for j in range(i+2, min(i+6, len(clean_cols))):
                    if re.match(r'^\d{1,2}$', clean_cols[j]) and clean_cols[j] != seed:
                        score = clean_cols[j]
                archer_entries.append({
                    'seed': seed,
                    'name': name,
                    'club': club,
                    'score': score
                })
    
    # Pair adjacent archers into matches
    matches = []
    for i in range(0, len(archer_entries) - 1, 2):
        a1 = archer_entries[i]
        a2 = archer_entries[i+1]
        matches.append({
            'seed1': a1['seed'],
            'archer1': a1['name'],
            'club1': a1['club'],
            'score1': a1['score'] or '6',
            'seed2': a2['seed'],
            'archer2': a2['name'],
            'club2': a2['club'],
            'score2': a2['score'] or '0',
            'sets1': '',
            'sets2': ''
        })
    
    return matches

print("Testing parser on all 4 categories...")
pages = [
    ('Standard Bow SD Prestasi Men - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESDM.php'),
    ('Standard Bow SD Prestasi Women - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESDW.php'),
    ('Standard Bow SMP Prestasi Men - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESMPM.php'),
    ('Standard Bow SMP Prestasi Women - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESMPW.php'),
]

new_brackets = {}
for cat_name, url in pages:
    matches = parse_ianseo_bracket_page(url)
    print(f"{cat_name} -> {len(matches)} matches parsed")
    new_brackets[cat_name] = [
        {
            'phase': '1/8 Eliminations & Finals',
            'matches': matches
        }
    ]

# Connect and update DB
conn = pymysql.connect(host='151.243.222.93', port=30036, user='ichsan', password='12345', database='archeris')
cur = conn.cursor(pymysql.cursors.DictCursor)
cur.execute("SELECT id, data_json FROM tournament_externals WHERE id = 53")
row = cur.fetchone()
dj = json.loads(row['data_json'])
dj['brackets'] = new_brackets

cur.execute("UPDATE tournament_externals SET data_json = %s WHERE id = 53", (json.dumps(dj, ensure_ascii=False),))
conn.commit()
print("Updated database for tournament ID 53 (Festival Indonesia Memanah 9) successfully!")
