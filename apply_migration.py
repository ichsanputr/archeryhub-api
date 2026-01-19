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
    
    with open('database/migrations/007_update_events_table.sql', 'r') as f:
        sql = f.read()
        
    # Split by semicolon if there are multiple statements
    for statement in sql.split(';'):
        if statement.strip():
            cursor.execute(statement)
            
    conn.commit()
    print("Migration applied successfully")
    conn.close()
except Exception as e:
    print(f"Error: {e}")
