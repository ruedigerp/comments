## 🗑️ **Redis-CLI Befehle:**

### **1. Einzelnen Kommentar löschen (z.B. ID 7):**

```bash
redis-cli
127.0.0.1:6379> DEL comments/7/text comments/7/created_at comments/7/mailaddress comments/7/post_id comments/7/username comments/7/active
```

### **2. Mit Pattern-Matching (eleganter):**

```bash
# Alle Keys für Kommentar 7 finden und löschen
127.0.0.1:6379> EVAL "return redis.call('del', unpack(redis.call('keys', ARGV[1])))" 0 comments/7/*

# Oder mit redis-cli --eval:
redis-cli --eval - comments/7/* <<< "return redis.call('del', unpack(redis.call('keys', ARGV[1])))"
```

### **3. Alle Kommentare löschen:**

```bash
# VORSICHT: Löscht alle Kommentar-Keys!
127.0.0.1:6379> EVAL "return redis.call('del', unpack(redis.call('keys', ARGV[1])))" 0 comments/*

# Auch den Counter zurücksetzen:
127.0.0.1:6379> DEL comment_counter
```

### **4. Specific Pattern (sicherer):**

```bash
# Nur Text-Keys löschen
127.0.0.1:6379> EVAL "return redis.call('del', unpack(redis.call('keys', ARGV[1])))" 0 comments/*/text

# Nur Kommentare für bestimmte Post-ID
127.0.0.1:6379> KEYS comments/*/post_id
# Dann manuell die IDs checken und löschen
```


## 🎯 **Schnelle Redis-CLI Befehle:**

### **Kommentar 7 komplett löschen:**

```bash
redis-cli DEL comments/7/text comments/7/created_at comments/7/mailaddress comments/7/post_id comments/7/username comments/7/active
```

### **Alle Kommentare löschen:**

```bash
redis-cli EVAL "return redis.call('del', unpack(redis.call('keys', 'comments/*')))" 0
redis-cli DEL comment_counter
```

### **Nur inaktive Kommentare löschen:**

```bash
# Finde alle active-Keys mit Wert "false"
redis-cli --scan --pattern "comments/*/active" | xargs redis-cli MGET

# Dann manuell die IDs der "false" Kommentare löschen
```

## 🔧 **Web-Interface Option:**

Wenn du die Admin-Endpunkte hinzufügst, kannst du über das Admin Panel löschen:

```bash
# Alle Kommentare löschen
curl -X DELETE "https://comments.kuepper.nrw/api/admin/delete-all-comments" \
  -H "Authorization: Bearer dein-admin-token"

# Kommentare für bestimmten Post löschen  
curl -X DELETE "https://comments.kuepper.nrw/api/admin/delete-comments-by-post?post_id=test-post" \
  -H "Authorization: Bearer dein-admin-token"

# Redis-Statistiken anzeigen
curl "https://comments.kuepper.nrw/api/admin/redis-stats" \
  -H "Authorization: Bearer dein-admin-token"
```

Der einfachste Weg für Kommentar 7 ist:

```bash
redis-cli DEL comments/7/text comments/7/created_at comments/7/mailaddress comments/7/post_id comments/7/username comments/7/active
```

Das löscht alle 6 Keys mit einem Befehl! 🗑️​​​​​​​​​​​​​​​​