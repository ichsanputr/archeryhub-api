import urllib.request, re

for to_id, name in [(16829, 'FAST Satria Pandhita 2024'), (16298, 'Kejurprov Jatim KU 2023')]:
    url = f'https://ianseo.net/Details.php?toId={to_id}'
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    try:
        with urllib.request.urlopen(req, timeout=10) as res:
            html = res.read().decode('utf-8', errors='ignore')
            has_elim = bool(re.search(r'Bracket|Elimination|Final', html, re.I))
            print(f"toId={to_id} ({name}):")
            print(f"  Has Bracket/Elimination text in HTML: {has_elim}")
            links = re.findall(r'href=[\'"]([^\'"]*[\'"])', html)
            rel_links = [l for l in links if any(k in l.lower() for k in ['bracket', 'elim', 'match'])]
            print(f"  Bracket-related links found: {rel_links}")
    except Exception as e:
        print(f"toId={to_id} Error: {e}")
