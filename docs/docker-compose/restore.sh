# --- restore.sh - Daten wiederherstellen ---
#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_file.rdb>"
    echo "Available backups:"
    ls -la backups/*.rdb 2>/dev/null || echo "No backups found"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "🔄 Restoring from backup: $BACKUP_FILE"

# Stop services
docker-compose down

# Copy backup to Redis container
docker-compose up -d valkey
sleep 5
docker cp "$BACKUP_FILE" $(docker-compose ps -q valkey):/data/dump.rdb

# Restart Redis to load data
docker-compose restart valkey
sleep 5

# Start application
docker-compose up -d comment-api

echo "✅ Restore completed"

