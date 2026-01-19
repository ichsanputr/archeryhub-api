import mysql.connector
import os
from dotenv import load_dotenv

load_dotenv()

try:
    conn = mysql.connector.connect(
        host=os.getenv("DB_HOST", "localhost"),
        user=os.getenv("DB_USER", "ichsan"),
        password=os.getenv("DB_PASSWORD", "12345"),
        database=os.getenv("DB_NAME", "archeryhub"),
        port=int(os.getenv("DB_PORT", "30036"))
    )
    cursor = conn.cursor()
    event_id = "d584d4ed-52ee-4aa5-b4c8-5f821df884ff"
    
    cursor.execute("SELECT start_date, end_date, registration_deadline FROM events WHERE id = %s", (event_id,))
    row = cursor.fetchone()
    if row:
        print(f"Start Date: {row[0]}")
        print(f"End Date: {row[1]}")
        print(f"Reg Deadline: {row[2]}")
    else:
        print("Event not found")
        
    conn.close()
except Exception as e:
    print(f"Error: {e}")
