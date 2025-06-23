# --- backup.sh - Daten-Backup ---
#!/bin/bash

BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "💾 Creating backup..."

mkdir -p "$BACKUP_DIR"

# Redis Daten exportieren
echo "📦 Backing up Redis data..."
docker-compose exec valkey redis-cli --rdb /tmp/dump.rdb
docker cp $(docker-compose ps -q valkey):/tmp/dump.rdb "$BACKUP_DIR/redis_$TIMESTAMP.rdb"

echo "✅ Backup created: $BACKUP_DIR/redis_$TIMESTAMP.rdb"
