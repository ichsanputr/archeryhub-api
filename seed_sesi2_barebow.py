import uuid
import random

# Configuration
EVENT_ID = "f9272ae0-f76f-11f0-87db-c3c8a1ce2650"
CATEGORY_ID = "0748ddc4-f832-11f0-87db-c3c8a1ce2650"
SESI_1_UUID = "218e5c30-f540-420e-935f-357a89dfb7f3"

# 1. Create Sesi 2
sesi_2_uuid = str(uuid.uuid4())
print(f"INSERT INTO qualification_sessions (uuid, event_category_uuid, session_name, session_order, status, created_at, updated_at) VALUES ('{sesi_2_uuid}', '{CATEGORY_ID}', 'Sesi 2', 2, 'draft', NOW(), NOW());")

# 2. Existing participants (unassigned)
existing_unassigned = [
    "79e42d91-f824-11f0-87db-c3c8a1ce2650",
    "ab99195e-dd68-482b-834b-147f18df2486",
    "ce76d947-f821-11f0-87db-c3c8a1ce2650"
]
existing_assigned = "65ccc306-25c6-4e00-aed6-ad2bc9f9825c"

# 3. Seed 16 additional archers
new_archers = []
for i in range(16):
    archer_uuid = str(uuid.uuid4())
    participant_uuid = str(uuid.uuid4())
    username = f"barebow_archer_{i+5}"
    email = f"{username}@example.com"
    full_name = f"Barebow Pro {i+5}"
    
    print(f"INSERT INTO archers (uuid, username, email, full_name, athlete_code, gender, date_of_birth, phone, address, experience_years, status, role, password, created_at, updated_at) VALUES ('{archer_uuid}', '{username}', '{email}', '{full_name}', 'PRO-B-{i+1000}', 'male', '1990-01-01', '08123456789', 'Jl. Panahan No. {i+10}', {random.randint(1,10)}, 'active', 'archer', 'password123', NOW(), NOW());")
    print(f"INSERT INTO event_participants (uuid, event_id, archer_id, category_id, payment_status, accreditation_status, registration_date, created_at, updated_at) VALUES ('{participant_uuid}', '{EVENT_ID}', '{archer_uuid}', '{CATEGORY_ID}', 'lunas', 'approved', NOW(), NOW(), NOW());")
    
    new_archers.append(participant_uuid)

# 4. Target Assignments
def create_assignments(session_uuid, participant_list, session_name):
    target_num = 1
    pos_idx = 0
    positions = ['A', 'B', 'C', 'D']
    
    for p_uuid in participant_list:
        # Skip the one already assigned in db manually if it's Target 1A in Sesi 1
        if session_name == "Sesi 1" and p_uuid == existing_assigned:
            pos_idx += 1 # Already handled by existing data
            continue
            
        print(f"INSERT INTO qualification_assignments (uuid, session_uuid, participant_uuid, target_number, target_position, created_at, updated_at) VALUES ('{uuid.uuid4()}', '{session_uuid}', '{p_uuid}', {target_num}, '{positions[pos_idx]}', NOW(), NOW());")
        
        pos_idx += 1
        if pos_idx >= 4:
            pos_idx = 0
            target_num += 1

# Sesi 1 pool: 1 existing assigned + 3 existing unassigned + 6 new
sesi1_pool = [existing_assigned] + existing_unassigned + new_archers[:6]
# Sesi 2 pool: 10 remaining new
sesi2_pool = new_archers[6:]

create_assignments(SESI_1_UUID, sesi1_pool, "Sesi 1")
create_assignments(sesi_2_uuid, sesi2_pool, "Sesi 2")

# 5. Target Cards (Target 1, 2, 3 for both sessions)
for session in [SESI_1_UUID, sesi_2_uuid]:
    for t_num in [1, 2, 3]:
        # check if it exists for Sesi 1
        print(f"INSERT IGNORE INTO target_cards (uuid, session_uuid, target_number, card_name, phase, status, created_at, updated_at) VALUES ('{uuid.uuid4()}', '{session}', {t_num}, 'Target {t_num}', 'qualification', 'active', NOW(), NOW());")
