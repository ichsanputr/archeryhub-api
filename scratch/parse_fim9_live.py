import urllib.request, re, json

pages = [
    ('Standard Bow SD Prestasi Men - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESDM.php'),
    ('Standard Bow SD Prestasi Women - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESDW.php'),
    ('Standard Bow SMP Prestasi Men - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESMPM.php'),
    ('Standard Bow SMP Prestasi Women - Ind. Finals', 'https://ianseo.net/TourData/2023/15863/IBESMPW.php'),
]

for cat_name, url in pages:
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    try:
        with urllib.request.urlopen(req) as res:
            html = res.read().decode('utf-8', errors='ignore')
            rows = re.findall(r'<tr[^>]*>(.*?)</tr>', html, re.S)
            archers_found = []
            for r in rows:
                cols = re.findall(r'<td[^>]*>(.*?)</td>', r, re.S)
                clean_cols = [re.sub(r'<[^>]+>', '', c).strip() for c in cols]
                clean_non_empty = [c for c in clean_cols if c and c != '&nbsp;']
                if len(clean_non_empty) >= 2 and any(re.search(r'[a-zA-Z]{3,}', c) for c in clean_non_empty):
                    archers_found.append(clean_non_empty)
            print(f"Category: {cat_name}")
            print(f"  Total Archer/Match Rows found in Ianseo table: {len(archers_found)}")
            for a in archers_found[:4]:
                print(f"    {a}")
    except Exception as e:
        print(f"Category: {cat_name} Error: {e}")
