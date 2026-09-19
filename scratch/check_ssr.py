import urllib.request, re

url = "http://localhost:3003/tournaments/kejuaraan-panahan-indoor-piala-komandan-yonif-315garuda"
try:
    with urllib.request.urlopen(url) as res:
        html = res.read().decode('utf-8', errors='ignore')
        print("Status:", res.status)
        print("Has brackets section:", "id=\"brackets\"" in html)
        print("Has 'Muhammad Umar':", "Muhammad Umar" in html or "MUHAMMAD UMAR" in html)
        print("Has 'Bagan Eliminasi':", "Bagan Eliminasi" in html or "Elimination Brackets" in html)
        
        matches = re.findall(r'<h2[^>]*>(.*?)</h2>', html)
        print("H2 headers:", matches)
except Exception as e:
    print("Error:", e)
