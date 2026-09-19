import urllib.request, re, json

url = "https://ianseo.net/Details.php?toId=15863"
req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
try:
    with urllib.request.urlopen(req) as res:
        html = res.read().decode('utf-8', errors='ignore')
        matches = re.findall(r'<a[^>]*href=[\'"]([^\'"]*)[\'"][^>]*>(.*?)</a>', html, re.I)
        print("All links on toId=15863:")
        for href, text in matches:
            clean_text = re.sub(r'<[^>]+>', '', text).strip()
            if any(k in href.lower() or k in clean_text.lower() for k in ['bracket', 'elim', 'final', 'standings', 'details']):
                print(f"  {href} --> {clean_text}")
except Exception as e:
    print("Error:", e)
