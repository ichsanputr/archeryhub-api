import mysql.connector
from mysql.connector import Error

# Remote Database Configuration
REMOTE_CONFIG = {
    'host': '151.243.222.93',
    'port': 30036,
    'user': 'ichsan',
    'password': '12345',
    'database': 'archeryhub'
}

# Local Database Configuration
LOCAL_CONFIG = {
    'host': 'localhost',
    'user': 'root',
    'password': '',
    'database': 'archeryhub'
}

def migrate():
    remote_conn = None
    local_conn = None
    
    try:
        # 1. Connect to Local
        print("Connecting to local database...")
        local_conn = mysql.connector.connect(**LOCAL_CONFIG)
        local_cursor = local_conn.cursor()
        
        # 2. Connect to Remote
        print(f"Connecting to remote database at {REMOTE_CONFIG['host']}...")
        remote_conn = mysql.connector.connect(
            host=REMOTE_CONFIG['host'],
            port=REMOTE_CONFIG['port'],
            user=REMOTE_CONFIG['user'],
            password=REMOTE_CONFIG['password']
        )
        remote_cursor = remote_conn.cursor()
        
        # 3. Create Remote DB if not exists
        print(f"Ensuring remote database '{REMOTE_CONFIG['database']}' exists...")
        remote_cursor.execute(f"CREATE DATABASE IF NOT EXISTS {REMOTE_CONFIG['database']}")
        remote_cursor.execute(f"USE {REMOTE_CONFIG['database']}")
        
        # 4. Get all tables from local
        local_cursor.execute("SHOW TABLES")
        tables = [t[0] for t in local_cursor.fetchall()]
        print(f"Found {len(tables)} tables to migrate: {', '.join(tables)}")
        
        # Disable foreign key checks for migration
        remote_cursor.execute("SET FOREIGN_KEY_CHECKS = 0")
        
        for table in tables:
            print(f"\nMigrating table: {table}")
            
            # 5. Get Create Table statement from local
            local_cursor.execute(f"SHOW CREATE TABLE {table}")
            create_stmt = local_cursor.fetchone()[1]
            
            # 6. Recreate table remotely
            remote_cursor.execute(f"DROP TABLE IF EXISTS {table}")
            remote_cursor.execute(create_stmt)
            print(f"  - Table structure recreated")
            
            # 7. Fetch data from local
            local_cursor.execute(f"SELECT * FROM {table}")
            rows = local_cursor.fetchall()
            
            if not rows:
                print(f"  - No data to migrate")
                continue
                
            # 8. Prepare Insert statement
            placeholders = ', '.join(['%s'] * len(rows[0]))
            columns = ', '.join([f"`{desc[0]}`" for desc in local_cursor.description])
            insert_query = f"INSERT INTO `{table}` ({columns}) VALUES ({placeholders})"
            
            # 9. Insert data remotely in chunks
            chunk_size = 500
            for i in range(0, len(rows), chunk_size):
                chunk = rows[i:i + chunk_size]
                remote_cursor.executemany(insert_query, chunk)
            
            remote_conn.commit()
            print(f"  - {len(rows)} records migrated successfully")
            
        # Re-enable foreign key checks
        remote_cursor.execute("SET FOREIGN_KEY_CHECKS = 1")
        print("\nMigration to remote completed successfully!")

    except Error as e:
        print(f"Error: {e}")
    finally:
        if remote_conn and remote_conn.is_connected():
            remote_cursor.close()
            remote_conn.close()
        if local_conn and local_conn.is_connected():
            local_cursor.close()
            local_conn.close()

if __name__ == "__main__":
    migrate()
