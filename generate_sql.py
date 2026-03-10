import json
import uuid

with open(r'C:\Users\HP\.gemini\antigravity\brain\90bc8360-bca8-4579-86a8-7f48609427d1\.system_generated\steps\8804\output.txt') as f:
    data = json.load(f)['result']

sql = []
for row in data:
    end_uuid = row['uuid']
    total = row['end_total']
    
    # Simple division into 3 arrows
    base = total // 3
    rem = total % 3
    arrows = [base] * 3
    for i in range(rem):
        arrows[i] += 1
        
    for i, a in enumerate(arrows):
        arrow_uuid = str(uuid.uuid4())
        sql.append(f"('{arrow_uuid}', '{end_uuid}', {i+1}, {a}, 0)")

with open('fill_arrows.sql', 'w') as f:
    for i in range(0, len(sql), 50):
        chunk = sql[i:i+50]
        f.write("INSERT INTO elimination_match_arrow_scores (uuid, match_end_uuid, arrow_no, score, is_x) VALUES\n")
        f.write(",\n".join(chunk))
        f.write(";\n")
