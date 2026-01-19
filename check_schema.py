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
    cursor.execute("DESCRIBE payment_transactions")
    for row in cursor.fetchall():
        print(row)
    conn.close()
except Exception as e:
    print(f"Error: {e}")
