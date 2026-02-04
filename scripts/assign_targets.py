import mysql.connector
import uuid
import random

def get_db_connection():
    return mysql.connector.connect(
        host='localhost',
        user='ichsan',
        password='12345',
        database='archeryhub',
        port=3306
    )

def assign_random_targets():
    conn = get_db_connection()
    cursor = conn.cursor(dictionary=True)

    event_id = '7247378c-b3cb-46d7-9ea3-78526733e7a7'
    category_id = '997f6d75-7f2b-40cf-a474-df4fafec8565' 
    session_id = '1489f482-b407-4ab8-b0a1-a1a8521f1165'

    # 1. Get Archers
    cursor.execute("""
        SELECT archer_id 
        FROM event_participants 
        WHERE event_id = %s AND category_id = %s AND status = 'Terdaftar'
    """, (event_id, category_id))
    archers = cursor.fetchall()
    
    # 2. Get Targets
    cursor.execute("""
        SELECT uuid, target_name 
        FROM event_targets 
        WHERE event_uuid = %s AND status = 'active'
    """, (event_id,))
    targets = cursor.fetchall()
    
    if len(archers) > len(targets):
        print(f"Error: Not enough targets ({len(targets)}) for archers ({len(archers)})")
        return

    random.shuffle(targets)
    
    # 3. Assign
    for i, archer in enumerate(archers):
        target = targets[i]
        assignment_id = str(uuid.uuid4())
        
        # In this system, target_position in qualification_target_assignments 
        # should match the target_name (letter) from event_targets for consistency
        cursor.execute("""
            INSERT INTO qualification_target_assignments (uuid, session_uuid, archer_uuid, target_uuid, target_position)
            VALUES (%s, %s, %s, %s, %s)
        """, (assignment_id, session_id, archer['archer_id'], target['uuid'], target['target_name']))
        
        print(f"Assigned archer {archer['archer_id']} to target UUID {target['uuid']} (Position {target['target_name']})")

    conn.commit()
    cursor.close()
    conn.close()
    print(f"Successfully assigned {len(archers)} archers to unique targets!")

if __name__ == "__main__":
    assign_random_targets()
