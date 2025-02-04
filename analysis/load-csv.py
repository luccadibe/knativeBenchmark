import sqlite3
import csv
import argparse

parser = argparse.ArgumentParser(description='Load CSV data into SQLite database')
parser.add_argument('--db', required=True, help='Path to SQLite database file')
parser.add_argument('--csv', required=True, help='Path to CSV data file')
args = parser.parse_args()

conn = sqlite3.connect(args.db)
cursor = conn.cursor()
cursor.execute("""
CREATE TABLE IF NOT EXISTS events (
    event_id INTEGER,
    timestamp TEXT
)
""")

with open(args.csv, "r") as file:
    reader = csv.DictReader(file)
    cursor.executemany("INSERT INTO events (event_id, timestamp) VALUES (?, ?)", 
                       [(row["event_id"], row["timestamp"]) for row in reader])

conn.commit()
conn.close()
