import re
import mysql.connector

# Database connection details
DB_CONFIG = {
    'host': '151.243.222.93',
    'port': 30036,
    'user': 'ichsan',
    'password': '12345',
    'database': 'archeryhub'
}

SCHEMA_FILE = 'schema.sql'

def parse_sql_schema(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Find all table blocks
    # We look for CREATE TABLE ... ( and then match balanced parentheses
    tables = {}
    
    # Simple split-based approach might be safer for SQL dumps
    chunks = re.split(r'CREATE TABLE (?:IF NOT EXISTS )?`(\w+)` \(', content, flags=re.IGNORECASE)
    
    # chunks[0] is header
    # chunks[1] is table1_name, chunks[2] is table1_content_plus_rest
    for i in range(1, len(chunks), 2):
        table_name = chunks[i]
        rest = chunks[i+1]
        
        # Find the closing ) of the CREATE TABLE statement
        # Since we use HeidiSQL/MySQL dump format, the main definition ends with ) ENGINE=...;
        # or just );
        end_match = re.search(r'\n\) ENGINE=.*?;', rest, re.IGNORECASE | re.DOTALL)
        if not end_match:
            end_match = re.search(r'\n\);', rest, re.IGNORECASE | re.DOTALL)
            
        if end_match:
            columns_block = rest[:end_match.start()]
            
            columns = set()
            for line in columns_block.split('\n'):
                line = line.strip()
                # Column lines MUST start with `Name`
                if line.startswith('`'):
                    col_match = re.match(r'^`(\w+)`', line)
                    if col_match:
                        columns.add(col_match.group(1))
            tables[table_name] = columns
            
    return tables

def get_db_schema(config):
    try:
        conn = mysql.connector.connect(**config)
        cursor = conn.cursor()
        
        cursor.execute("SHOW TABLES")
        tables = [row[0] for row in cursor.fetchall()]
        
        db_schema = {}
        for table in tables:
            cursor.execute(f"DESCRIBE `{table}`")
            columns = set()
            for row in cursor.fetchall():
                columns.add(row[0])
            db_schema[table] = columns
            
        cursor.close()
        conn.close()
        return db_schema
    except mysql.connector.Error as err:
        print(f"Error: {err}")
        return None

def compare_schemas(expected, actual):
    report = []
    report.append("--- SCHEMA COMPARISON REPORT ---")
    
    expected_tables = set(expected.keys())
    actual_tables = set(actual.keys())
    
    missing_tables = expected_tables - actual_tables
    extra_tables = actual_tables - expected_tables
    
    if missing_tables:
        report.append(f"\n[!] Tables in schema.sql but MISSING in Database: {', '.join(sorted(missing_tables))}")
    if extra_tables:
        report.append(f"\n[+] Extra tables in Database (not in schema.sql): {', '.join(sorted(extra_tables))}")
        
    common_tables = expected_tables & actual_tables
    for table in sorted(common_tables):
        exp_cols = expected[table]
        act_cols = actual[table]
        
        missing_cols = exp_cols - act_cols
        extra_cols = act_cols - exp_cols
        
        if missing_cols or extra_cols:
            report.append(f"\nTable `{table}`:")
            if missing_cols:
                report.append(f"  [-] Missing columns in Prod: {', '.join(sorted(missing_cols))}")
            if extra_cols:
                report.append(f"  [+] Extra columns in Prod: {', '.join(sorted(extra_cols))}")
    
    if len(report) == 1:
        report.append("\nNo differences found between schema.sql and production database.")
        
    return "\n".join(report)

if __name__ == "__main__":
    expected_schema = parse_sql_schema(SCHEMA_FILE)
    actual_schema = get_db_schema(DB_CONFIG)
    
    if expected_schema and actual_schema:
        report = compare_schemas(expected_schema, actual_schema)
        # print(report)
        with open('schema_report.txt', 'w') as f:
            f.write(report)
    else:
        with open('schema_report.txt', 'w') as f:
            f.write("Failed to perform comparison because schemas could not be retrieved.")
