package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Comment stellt einen Kommentar dar
type Comment struct {
	ID          int    `json:"id"`
	PostID      string `json:"post_id"`
	Username    string `json:"username"`
	MailAddress string `json:"mailaddress"`
	Text        string `json:"text"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

// CommentService verwaltet Kommentare in ValKey
type CommentService struct {
	client *redis.Client
	ctx    context.Context
}

// AuthConfig hält die Authentifizierungskonfiguration
type AuthConfig struct {
	AdminToken string
	Enabled    bool
}

// Template-Daten Struktur
type JSWidgetTemplateData struct {
	ApiUrl  string
	Version string
	Stage   string
}

// Template Cache für bessere Performance
type TemplateCache struct {
	template     *template.Template
	lastModified time.Time
	mutex        sync.RWMutex
}

var jsTemplateCache = &TemplateCache{}

var (
	version   = "dev"         // Default-Wert falls nicht gesetzt
	stage     = "development" // Default-Wert falls nicht gesetzt
	startTime = time.Now().UTC()
)

// Helper-Funktionen für Environment-Variablen
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}

// TokenAuth erstellt eine neue Auth-Konfiguration
func NewTokenAuth() *AuthConfig {
	adminToken := getEnv("ADMIN_TOKEN", "")

	// Falls kein Token gesetzt ist, einen generieren und warnen
	if adminToken == "" {
		adminToken = generateRandomToken()
		log.Printf("⚠️  ADMIN_TOKEN not set! Generated temporary token: %s", adminToken)
		log.Println("🔐 Set ADMIN_TOKEN environment variable for production!")
	}

	enabled := getEnv("AUTH_ENABLED", "true") == "true"

	return &AuthConfig{
		AdminToken: adminToken,
		Enabled:    enabled,
	}
}

// generateRandomToken erstellt einen sicheren zufälligen Token
func generateRandomToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate random token:", err)
	}
	return hex.EncodeToString(bytes)
}

// AuthMiddleware schützt Endpunkte mit Token-Authentifizierung
func (auth *AuthConfig) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Falls Auth deaktiviert ist, durchlassen
		if !auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Token aus verschiedenen Quellen extrahieren
		token := extractToken(r)

		if token == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing authentication token")
			return
		}

		// Token validieren (constant-time comparison gegen timing attacks)
		if !auth.validateToken(token) {
			respondWithError(w, http.StatusUnauthorized, "Invalid authentication token")
			return
		}

		// Request durchlassen
		next.ServeHTTP(w, r)
	})
}

// extractToken extrahiert den Token aus verschiedenen Quellen
func extractToken(r *http.Request) string {
	// 1. Authorization Header: "Bearer <token>"
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 2. X-Admin-Token Header
	if token := r.Header.Get("X-Admin-Token"); token != "" {
		return token
	}

	// 3. Query Parameter (weniger sicher, nur für Tests)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}

// validateToken prüft den Token sicher
func (auth *AuthConfig) validateToken(token string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(auth.AdminToken)) == 1
}

// respondWithError sendet eine JSON-Fehlerantwort
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// NewCommentService erstellt einen neuen CommentService
func NewCommentService(redisAddr, redisPassword string, redisDB int) *CommentService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	return &CommentService{
		client: rdb,
		ctx:    context.Background(),
	}
}

// generateCommentID generiert eine neue Kommentar-ID
func (cs *CommentService) generateCommentID() (int, error) {
	id, err := cs.client.Incr(cs.ctx, "comment_counter").Result()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// CreateComment erstellt einen neuen Kommentar
func (cs *CommentService) CreateComment(postID, username, mailAddress, text string) (*Comment, error) {
	id, err := cs.generateCommentID()
	if err != nil {
		return nil, fmt.Errorf("fehler beim Generieren der ID: %w", err)
	}

	comment := &Comment{
		ID:          id,
		PostID:      postID,
		Username:    username,
		MailAddress: mailAddress,
		Text:        text,
		Active:      false, // Standardmäßig aktiv
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// Kommentar-Daten in ValKey speichern
	pipe := cs.client.Pipeline()
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/post_id", id), postID, 0)
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/username", id), username, 0)
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/mailaddress", id), mailAddress, 0)
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/text", id), text, 0)
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/active", id), comment.Active, 0)
	pipe.Set(cs.ctx, fmt.Sprintf("comments/%d/created_at", id), comment.CreatedAt, 0)

	_, err = pipe.Exec(cs.ctx)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Speichern des Kommentars: %w", err)
	}

	return comment, nil
}

// GetComment holt einen Kommentar anhand der ID
func (cs *CommentService) GetComment(id int) (*Comment, error) {
	pipe := cs.client.Pipeline()
	postIDCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/post_id", id))
	usernameCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/username", id))
	mailAddressCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/mailaddress", id))
	textCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/text", id))
	activeCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/active", id))
	createdAtCmd := pipe.Get(cs.ctx, fmt.Sprintf("comments/%d/created_at", id))

	_, err := pipe.Exec(cs.ctx)
	if err != nil {
		return nil, fmt.Errorf("kommentar nicht gefunden: %w", err)
	}

	postID, _ := postIDCmd.Result()
	username, _ := usernameCmd.Result()
	mailAddress, _ := mailAddressCmd.Result()
	text, _ := textCmd.Result()
	activeStr, _ := activeCmd.Result()
	createdAt, _ := createdAtCmd.Result()

	active := activeStr == "true"

	return &Comment{
		ID:          id,
		PostID:      postID,
		Username:    username,
		MailAddress: mailAddress,
		Text:        text,
		Active:      active,
		CreatedAt:   createdAt,
	}, nil
}

// UpdateCommentStatus aktualisiert den Active-Status eines Kommentars
func (cs *CommentService) UpdateCommentStatus(id int, active bool) error {
	activeStr := "false"
	if active {
		activeStr = "true"
	}

	err := cs.client.Set(cs.ctx, fmt.Sprintf("comments/%d/active", id), activeStr, 0).Err()
	if err != nil {
		return fmt.Errorf("fehler beim Aktualisieren des Status: %w", err)
	}

	return nil
}

// GetCommentsByPostID holt alle Kommentare für einen bestimmten Blog-Post
// GetCommentsByPostID holt alle Kommentare für einen bestimmten Blog-Post
func (cs *CommentService) GetCommentsByPostID(postID string, includeInactive bool) ([]*Comment, error) {
	log.Printf("Suche Kommentare für PostID: '%s'", postID)

	// Alle username Keys finden (als Indikator für existierende Kommentare)
	keys, err := cs.client.Keys(cs.ctx, "comments/*/username").Result()
	if err != nil {
		return nil, fmt.Errorf("fehler beim Abrufen der Kommentar-Keys: %w", err)
	}

	log.Printf("Gefundene Kommentar-Keys: %v", keys)

	var comments []*Comment
	for _, key := range keys {
		// ID aus dem Key extrahieren (comments/9/username -> 9)
		parts := strings.Split(key, "/")
		if len(parts) >= 2 {
			idStr := parts[1]
			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Printf("Fehler beim Parsen der ID %s: %v", idStr, err)
				continue
			}

			// Post-ID für diesen Kommentar abrufen
			postIDKey := fmt.Sprintf("comments/%d/post_id", id)
			storedPostID, err := cs.client.Get(cs.ctx, postIDKey).Result()
			if err != nil {
				log.Printf("Fehler beim Abrufen der PostID für Kommentar %d: %v", id, err)
				continue
			}

			log.Printf("Kommentar %d: Gespeicherte PostID='%s', Gesuchte PostID='%s'", id, storedPostID, postID)

			// Nur Kommentare für die gewünschte PostID
			if storedPostID == postID {
				comment, err := cs.GetComment(id)
				if err != nil {
					log.Printf("Fehler beim Laden des Kommentars %d: %v", id, err)
					continue
				}

				// Nur aktive Kommentare einschließen, außer explizit anders gewünscht
				if comment.Active || includeInactive {
					comments = append(comments, comment)
				}
			}
		}
	}

	log.Printf("Gefundene Kommentare für PostID '%s': %d", postID, len(comments))
	return comments, nil
}

// GetAllComments holt alle Kommentare (nur aktive standardmäßig)
func (cs *CommentService) GetAllComments(includeInactive bool) ([]*Comment, error) {
	// Alle Kommentar-Keys finden
	keys, err := cs.client.Keys(cs.ctx, "comments/*/username").Result()
	if err != nil {
		return nil, fmt.Errorf("fehler beim Abrufen der Kommentar-Keys: %w", err)
	}

	var comments []*Comment
	for _, key := range keys {
		// ID aus dem Key extrahieren
		parts := strings.Split(key, "/")
		if len(parts) >= 2 {
			idStr := parts[1]
			id, err := strconv.Atoi(idStr)
			if err != nil {
				continue
			}

			comment, err := cs.GetComment(id)
			if err != nil {
				continue
			}

			// Nur aktive Kommentare einschließen, außer explizit anders gewünscht
			if comment.Active || includeInactive {
				comments = append(comments, comment)
			}
		}
	}

	return comments, nil
}

// DeleteComment löscht einen Kommentar
func (cs *CommentService) DeleteComment(id int) error {
	pipe := cs.client.Pipeline()
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/post_id", id))
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/username", id))
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/mailaddress", id))
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/text", id))
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/active", id))
	pipe.Del(cs.ctx, fmt.Sprintf("comments/%d/created_at", id))

	_, err := pipe.Exec(cs.ctx)
	if err != nil {
		return fmt.Errorf("fehler beim Löschen des Kommentars: %w", err)
	}

	return nil
}

// HTTP Handler
type CommentHandler struct {
	service *CommentService
}

func NewCommentHandler(service *CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

func (h *CommentHandler) CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PostID      string `json:"post_id"`
		Username    string `json:"username"`
		MailAddress string `json:"mailaddress"`
		Text        string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültige JSON", http.StatusBadRequest)
		return
	}

	// Validierung
	if req.PostID == "" || req.Username == "" || req.MailAddress == "" || req.Text == "" {
		http.Error(w, "Alle Felder sind erforderlich", http.StatusBadRequest)
		return
	}

	comment, err := h.service.CreateComment(req.PostID, req.Username, req.MailAddress, req.Text)
	if err != nil {
		http.Error(w, "Fehler beim Erstellen des Kommentars", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func (h *CommentHandler) GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("post_id")
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	// includeInactive := true

	var comments []*Comment
	var err error

	if postID != "" {
		// Kommentare für einen bestimmten Post abrufen
		comments, err = h.service.GetCommentsByPostID(postID, includeInactive)
	} else {
		// Alle Kommentare abrufen
		comments, err = h.service.GetAllComments(includeInactive)
	}

	if err != nil {
		http.Error(w, "Fehler beim Abrufen der Kommentare", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (h *CommentHandler) GetCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Ungültige ID", http.StatusBadRequest)
		return
	}

	comment, err := h.service.GetComment(id)
	if err != nil {
		http.Error(w, "Kommentar nicht gefunden", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func (h *CommentHandler) UpdateCommentStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Ungültige ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Active bool `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültige JSON", http.StatusBadRequest)
		return
	}

	err = h.service.UpdateCommentStatus(id, req.Active)
	if err != nil {
		http.Error(w, "Fehler beim Aktualisieren des Status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Status aktualisiert"})
}

func (h *CommentHandler) DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Ungültige ID", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteComment(id)
	if err != nil {
		http.Error(w, "Fehler beim Löschen des Kommentars", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Kommentar gelöscht"})
}

// AdminPanelHandler serviert das Admin-Panel HTML
func (h *CommentHandler) AdminPanelHandler(w http.ResponseWriter, r *http.Request) {
	htmlContent := `<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Comment Admin Panel</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }

        .header {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }

        .header h1 {
            font-size: 2rem;
            margin-bottom: 10px;
        }

        .auth-section {
            padding: 20px 30px;
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
        }

        .auth-input {
            display: flex;
            gap: 10px;
            align-items: center;
            flex-wrap: wrap;
        }

        .auth-input input {
            flex: 1;
            min-width: 300px;
            padding: 10px 15px;
            border: 2px solid #e1e5e9;
            border-radius: 8px;
            font-size: 14px;
        }

        .auth-input input:focus {
            outline: none;
            border-color: #4facfe;
        }

        .btn {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.3s ease;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(79, 172, 254, 0.3);
        }

        .btn:disabled {
            background: #6c757d;
            cursor: not-allowed;
            transform: none;
            box-shadow: none;
        }

        .stats {
            padding: 20px 30px;
            background: #f8f9fa;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 20px;
        }

        .stat-card {
            background: white;
            padding: 15px;
            border-radius: 10px;
            text-align: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }

        .stat-number {
            font-size: 2rem;
            font-weight: bold;
            color: #4facfe;
        }

        .stat-label {
            color: #666;
            font-size: 0.9rem;
            margin-top: 5px;
        }

        .filters {
            padding: 20px 30px;
            background: white;
            border-bottom: 1px solid #e9ecef;
            display: flex;
            gap: 15px;
            align-items: center;
            flex-wrap: wrap;
        }

        .filter-group {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .filter-group select {
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
        }

        .comments-list {
            padding: 30px;
            max-height: 600px;
            overflow-y: auto;
        }

        .comment-item {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 20px;
            transition: all 0.3s ease;
            position: relative;
        }

        .comment-item:hover {
            transform: translateX(5px);
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
        }

        .comment-item.inactive {
            background: #f8f9fa;
            opacity: 0.7;
            border-left: 4px solid #dc3545;
        }

        .comment-item.active {
            border-left: 4px solid #28a745;
        }

        .comment-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 15px;
            flex-wrap: wrap;
            gap: 10px;
        }

        .comment-meta {
            flex: 1;
        }

        .comment-author {
            font-weight: 600;
            color: #333;
            font-size: 1.1rem;
        }

        .comment-email {
            color: #666;
            font-size: 0.9rem;
            margin-top: 2px;
        }

        .comment-post-id {
            background: #e3f2fd;
            color: #1976d2;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.8rem;
            margin-top: 5px;
            display: inline-block;
        }

        .comment-actions {
            display: flex;
            gap: 10px;
            align-items: center;
        }

        .status-toggle {
            background: none;
            border: 2px solid;
            padding: 8px 16px;
            border-radius: 20px;
            cursor: pointer;
            font-weight: 500;
            transition: all 0.3s ease;
            font-size: 0.9rem;
        }

        .status-toggle.active {
            border-color: #28a745;
            color: #28a745;
            background: rgba(40, 167, 69, 0.1);
        }

        .status-toggle.inactive {
            border-color: #dc3545;
            color: #dc3545;
            background: rgba(220, 53, 69, 0.1);
        }

        .status-toggle:hover {
            transform: scale(1.05);
        }

        .comment-text {
            color: #444;
            line-height: 1.6;
            margin-bottom: 15px;
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            border-left: 3px solid #4facfe;
        }

        .comment-footer {
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 0.9rem;
            color: #666;
            padding-top: 10px;
            border-top: 1px solid #e9ecef;
        }

        .comment-date {
            display: flex;
            align-items: center;
            gap: 5px;
        }

        .comment-id {
            background: #6c757d;
            color: white;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.8rem;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
            font-size: 1.1rem;
        }

        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 30px;
            border: 1px solid #f5c6cb;
        }

        .success {
            background: #d4edda;
            color: #155724;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 30px;
            border: 1px solid #c3e6cb;
        }

        .empty-state {
            text-align: center;
            padding: 60px 30px;
            color: #666;
        }

        .empty-state h3 {
            margin-bottom: 10px;
            color: #333;
        }

        @media (max-width: 768px) {
            .container {
                margin: 10px;
                border-radius: 10px;
            }
            
            .auth-input {
                flex-direction: column;
                align-items: stretch;
            }
            
            .auth-input input {
                min-width: auto;
            }
            
            .comment-header {
                flex-direction: column;
                align-items: flex-start;
            }
            
            .comment-actions {
                width: 100%;
                justify-content: flex-start;
            }
            
            .stats {
                grid-template-columns: repeat(2, 1fr);
            }
        }

        .refresh-btn {
            background: #17a2b8;
            margin-left: auto;
        }

        .logout-btn {
            background: #dc3545 !important;
        }

        .auto-refresh {
            display: flex;
            align-items: center;
            gap: 8px;
            color: #666;
            font-size: 0.9rem;
        }

        .auto-refresh input[type="checkbox"] {
            transform: scale(1.2);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 Comment Admin Panel</h1>
            <p>Verwaltung aller Kommentare</p>
        </div>

        <div class="auth-section">
            <div class="auth-input">
                <input type="password" id="adminToken" placeholder="Admin Token eingeben">
                <button class="btn" onclick="authenticate()">🔑 Authentifizieren</button>
                <button class="btn refresh-btn" onclick="loadComments()" disabled id="refreshBtn">🔄 Aktualisieren</button>
                <button class="btn logout-btn" onclick="logout()">🚪 Abmelden</button>
                <div class="auto-refresh">
                    <input type="checkbox" id="autoRefresh" onchange="toggleAutoRefresh()">
                    <label for="autoRefresh">Auto-Refresh (30s)</label>
                </div>
            </div>
        </div>

        <div id="statsSection" class="stats" style="display: none;">
            <div class="stat-card">
                <div class="stat-number" id="totalComments">0</div>
                <div class="stat-label">Gesamt</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="activeComments">0</div>
                <div class="stat-label">Aktiv</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="inactiveComments">0</div>
                <div class="stat-label">Inaktiv</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="uniquePosts">0</div>
                <div class="stat-label">Posts</div>
            </div>
        </div>

        <div id="filtersSection" class="filters" style="display: none;">
            <div class="filter-group">
                <label>Status:</label>
                <select id="statusFilter" onchange="filterComments()">
                    <option value="all">Alle</option>
                    <option value="active">Nur Aktive</option>
                    <option value="inactive">Nur Inaktive</option>
                </select>
            </div>
            <div class="filter-group">
                <label>Post:</label>
                <select id="postFilter" onchange="filterComments()">
                    <option value="all">Alle Posts</option>
                </select>
            </div>
            <div class="filter-group">
                <label>Sortierung:</label>
                <select id="sortFilter" onchange="filterComments()">
                    <option value="newest">Neueste zuerst</option>
                    <option value="oldest">Älteste zuerst</option>
                </select>
            </div>
        </div>

        <div id="messageArea"></div>

        <div class="comments-list">
            <div id="commentsContainer">
                <div class="loading">Bitte authentifizieren Sie sich, um Kommentare zu laden</div>
            </div>
        </div>
    </div>

    <script>
        // Token-Management und Admin Panel JavaScript
        let adminToken = '';
        let allComments = [];
        let autoRefreshInterval = null;
        const API_BASE = '/api/comments';

        // Token aus verschiedenen Quellen laden
        function loadTokenFromStorage() {
            const urlParams = new URLSearchParams(window.location.search);
            const urlToken = urlParams.get('token');
            if (urlToken) {
                adminToken = urlToken;
                document.getElementById('adminToken').value = urlToken;
                window.history.replaceState({}, document.title, window.location.pathname);
                return urlToken;
            }

            const sessionToken = sessionStorage.getItem('adminToken');
            if (sessionToken) {
                adminToken = sessionToken;
                document.getElementById('adminToken').value = sessionToken;
                return sessionToken;
            }

            return null;
        }

        function saveTokenToStorage(token) {
            adminToken = token;
            sessionStorage.setItem('adminToken', token);
        }

        function clearTokenFromStorage() {
            adminToken = '';
            sessionStorage.removeItem('adminToken');
            document.getElementById('adminToken').value = '';
        }

        function showMessage(message, type = 'error') {
            const messageArea = document.getElementById('messageArea');
            messageArea.innerHTML = '<div class="' + type + '">' + message + '</div>';
            setTimeout(() => {
                messageArea.innerHTML = '';
            }, 5000);
        }

        function authenticate() {
            const token = document.getElementById('adminToken').value.trim();
            if (!token) {
                showMessage('Bitte geben Sie einen Admin Token ein');
                return;
            }
            
            saveTokenToStorage(token);
            enableAuthenticatedUI();
            loadComments();
        }

        function enableAuthenticatedUI() {
            document.getElementById('refreshBtn').disabled = false;
            const tokenInput = document.getElementById('adminToken');
            tokenInput.style.borderColor = '#28a745';
            tokenInput.style.backgroundColor = '#d4edda';
        }

        function disableAuthenticatedUI() {
            document.getElementById('refreshBtn').disabled = true;
            document.getElementById('statsSection').style.display = 'none';
            document.getElementById('filtersSection').style.display = 'none';
            
            const tokenInput = document.getElementById('adminToken');
            tokenInput.style.borderColor = '#dc3545';
            tokenInput.style.backgroundColor = '#f8d7da';
            
            document.getElementById('commentsContainer').innerHTML = 
                '<div class="loading">Bitte authentifizieren Sie sich, um Kommentare zu laden</div>';
        }

        function logout() {
            clearTokenFromStorage();
            disableAuthenticatedUI();
            
            if (autoRefreshInterval) {
                clearInterval(autoRefreshInterval);
                autoRefreshInterval = null;
                document.getElementById('autoRefresh').checked = false;
            }
            
            showMessage('Abgemeldet', 'success');
        }

        async function apiCall(endpoint, options = {}) {
            if (!adminToken) {
                showMessage('Nicht authentifiziert');
                disableAuthenticatedUI();
                return null;
            }

            const headers = {
                'Authorization': 'Bearer ' + adminToken,
                'Content-Type': 'application/json'
            };

            if (options.headers) {
                Object.assign(headers, options.headers);
            }

            try {
                const response = await fetch(endpoint, { 
                    ...options, 
                    headers: headers 
                });
                
                if (response.status === 401) {
                    showMessage('Token ungültig oder abgelaufen. Bitte neu anmelden.', 'error');
                    clearTokenFromStorage();
                    disableAuthenticatedUI();
                    return null;
                }
                
                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error('HTTP ' + response.status + ': ' + errorText);
                }

                return await response.json();
            } catch (error) {
                if (error.message.includes('401')) {
                    clearTokenFromStorage();
                    disableAuthenticatedUI();
                    showMessage('Authentifizierung fehlgeschlagen', 'error');
                } else {
                    showMessage('API Fehler: ' + error.message);
                }
                return null;
            }
        }

        async function loadComments() {
            if (!adminToken) {
                showMessage('Kein Token verfügbar');
                return;
            }

            document.getElementById('commentsContainer').innerHTML = '<div class="loading">Lade Kommentare...</div>';
            
            const adminInfo = await apiCall(API_BASE + '/admin/info');
            if (adminInfo) {
                updateStats(adminInfo);
                document.getElementById('statsSection').style.display = 'grid';
                document.getElementById('filtersSection').style.display = 'flex';
                enableAuthenticatedUI();
                showMessage('Erfolgreich authentifiziert', 'success');
            } else {
                return;
            }

            const comments = await apiCall(API_BASE + '?include_inactive=true');
            if (comments) {
                allComments = comments;
                updatePostFilter();
                filterComments();
                showMessage('Kommentare erfolgreich geladen', 'success');
            }
        }

        function updateStats(adminInfo) {
            document.getElementById('totalComments').textContent = adminInfo.total_comments;
            document.getElementById('activeComments').textContent = adminInfo.active_comments;
            document.getElementById('inactiveComments').textContent = adminInfo.inactive_comments;
            
            const uniquePosts = new Set(allComments.map(c => c.post_id)).size;
            document.getElementById('uniquePosts').textContent = uniquePosts;
        }

        function updatePostFilter() {
            const postFilter = document.getElementById('postFilter');
            const uniquePosts = [...new Set(allComments.map(c => c.post_id))].sort();
            
            postFilter.innerHTML = '<option value="all">Alle Posts</option>';
            uniquePosts.forEach(postId => {
                const option = document.createElement('option');
                option.value = postId;
                option.textContent = postId;
                postFilter.appendChild(option);
            });
        }

        function filterComments() {
            const statusFilter = document.getElementById('statusFilter').value;
            const postFilter = document.getElementById('postFilter').value;
            const sortFilter = document.getElementById('sortFilter').value;

            let filteredComments = [...allComments];

            if (statusFilter === 'active') {
                filteredComments = filteredComments.filter(c => c.active);
            } else if (statusFilter === 'inactive') {
                filteredComments = filteredComments.filter(c => !c.active);
            }

            if (postFilter !== 'all') {
                filteredComments = filteredComments.filter(c => c.post_id === postFilter);
            }

            filteredComments.sort((a, b) => {
                const dateA = new Date(a.created_at);
                const dateB = new Date(b.created_at);
                return sortFilter === 'newest' ? dateB - dateA : dateA - dateB;
            });

            displayComments(filteredComments);
        }

        function displayComments(comments) {
            const container = document.getElementById('commentsContainer');
            
            if (comments.length === 0) {
                container.innerHTML = '<div class="empty-state"><h3>Keine Kommentare gefunden</h3><p>Es gibt keine Kommentare, die den aktuellen Filtern entsprechen.</p></div>';
                return;
            }

            const commentsHTML = comments.map(comment => {
                const date = new Date(comment.created_at);
                const formattedDate = date.toLocaleDateString('de-DE', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit'
                });

                return '<div class="comment-item ' + (comment.active ? 'active' : 'inactive') + '">' +
                    '<div class="comment-header">' +
                        '<div class="comment-meta">' +
                            '<div class="comment-author">👤 ' + escapeHtml(comment.username) + '</div>' +
                            '<div class="comment-email">📧 ' + escapeHtml(comment.mailaddress) + '</div>' +
                            '<div class="comment-post-id">📝 ' + escapeHtml(comment.post_id) + '</div>' +
                        '</div>' +
                        '<div class="comment-actions">' +
                            '<button class="status-toggle ' + (comment.active ? 'active' : 'inactive') + '" onclick="toggleCommentStatus(' + comment.id + ', ' + !comment.active + ')">' +
                                (comment.active ? '✅ Aktiv' : '❌ Inaktiv') +
                            '</button>' +
                        '</div>' +
                    '</div>' +
                    '<div class="comment-text">' + escapeHtml(comment.text) + '</div>' +
                    '<div class="comment-footer">' +
                        '<div class="comment-date">🕒 ' + formattedDate + '</div>' +
                        '<div class="comment-id">ID: ' + comment.id + '</div>' +
                    '</div>' +
                '</div>';
            }).join('');

            container.innerHTML = commentsHTML;
        }

        async function toggleCommentStatus(commentId, newStatus) {
            const result = await apiCall(API_BASE + '/' + commentId + '/status', {
                method: 'PUT',
                body: JSON.stringify({ active: newStatus })
            });

            if (result) {
                const comment = allComments.find(c => c.id === commentId);
                if (comment) {
                    comment.active = newStatus;
                }
                
                showMessage('Kommentar ' + (newStatus ? 'aktiviert' : 'deaktiviert'), 'success');
                filterComments();
                
                const adminInfo = await apiCall(API_BASE + '/admin/info');
                if (adminInfo) {
                    updateStats(adminInfo);
                }
            }
        }

        function toggleAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');
            
            if (checkbox.checked) {
                autoRefreshInterval = setInterval(() => {
                    if (adminToken) {
                        loadComments();
                    }
                }, 30000);
                showMessage('Auto-Refresh aktiviert (30s)', 'success');
            } else {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                    autoRefreshInterval = null;
                }
                showMessage('Auto-Refresh deaktiviert', 'success');
            }
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Initialisierung
        document.addEventListener('DOMContentLoaded', function() {
            const savedToken = loadTokenFromStorage();
            if (savedToken) {
                enableAuthenticatedUI();
                loadComments();
            }

            document.getElementById('adminToken').addEventListener('keypress', function(e) {
                if (e.key === 'Enter') {
                    authenticate();
                }
            });

            window.addEventListener('beforeunload', function() {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                }
            });
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlContent))
}

func (h *CommentHandler) AdminInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Alle Kommentare inkl. inaktive laden
	allComments, err := h.service.GetAllComments(true)
	if err != nil {
		http.Error(w, "Fehler beim Abrufen der Statistiken", http.StatusInternalServerError)
		return
	}

	// Statistiken berechnen
	activeCount := 0
	inactiveCount := 0
	postIds := make(map[string]int)
	recentComments := 0

	// 24 Stunden ago
	yesterday := time.Now().Add(-24 * time.Hour)

	for _, comment := range allComments {
		if comment.Active {
			activeCount++
		} else {
			inactiveCount++
		}

		// Posts zählen
		postIds[comment.PostID]++

		// Recent comments (letzte 24h)
		if createdAt, err := time.Parse(time.RFC3339, comment.CreatedAt); err == nil {
			if createdAt.After(yesterday) {
				recentComments++
			}
		}
	}
	// Top Posts (Posts mit den meisten Kommentaren)
	type PostStats struct {
		PostID       string `json:"post_id"`
		CommentCount int    `json:"comment_count"`
	}

	var topPosts []PostStats
	for postID, count := range postIds {
		topPosts = append(topPosts, PostStats{
			PostID:       postID,
			CommentCount: count,
		})
	}

	// Nach Anzahl sortieren (Top 5)
	for i := 0; i < len(topPosts)-1; i++ {
		for j := i + 1; j < len(topPosts); j++ {
			if topPosts[j].CommentCount > topPosts[i].CommentCount {
				topPosts[i], topPosts[j] = topPosts[j], topPosts[i]
			}
		}
	}

	if len(topPosts) > 5 {
		topPosts = topPosts[:5]
	}

	response := map[string]interface{}{
		"total_comments":    len(allComments),
		"active_comments":   activeCount,
		"inactive_comments": inactiveCount,
		"unique_posts":      len(postIds),
		"recent_comments":   recentComments, // Letzte 24h
		"top_posts":         topPosts,
		"server_time":       time.Now().UTC().Format(time.RFC3339),
		"version":           version,
		"stage":             stage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Static File Handler für JavaScript und andere Assets
func setupStaticRoutesx(r *mux.Router) {
	// Spezifische JS-Route mit korrektem Handler
	r.PathPrefix("/js/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveJavaScript(w, r)
	})

	// CSS-Route
	r.PathPrefix("/css/").Handler(
		http.StripPrefix("/css/", addCSSHeaders(http.FileServer(http.Dir("./static/css/")))))

	// Allgemeine Static Files
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", addStaticHeaders(http.FileServer(http.Dir("./static/")))))
}

func setupStaticRoutes(r *mux.Router) {
	// Custom Static Handler mit korrekten Content-Types
	staticHandler := http.StripPrefix("/static/",
		addStaticHeaders(http.FileServer(http.Dir("./static/"))))

	// JavaScript Handler mit korrektem MIME-Type
	jsHandler := http.StripPrefix("/js/",
		addJavaScriptHeaders(http.FileServer(http.Dir("./static/js/"))))

	// CSS Handler
	cssHandler := http.StripPrefix("/css/",
		addCSSHeaders(http.FileServer(http.Dir("./static/css/"))))

	// Routes registrieren
	r.PathPrefix("/js/").Handler(jsHandler)
	r.PathPrefix("/css/").Handler(cssHandler)
	r.PathPrefix("/static/").Handler(staticHandler)
}

func addJavaScriptHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Richtiger MIME-Type für JavaScript
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

		// CORS für JavaScript Files
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Caching (1 Stunde)
		w.Header().Set("Cache-Control", "public, max-age=3600")

		// Security Headers (OHNE nosniff für JS)
		// w.Header().Set("X-Content-Type-Options", "nosniff") // ENTFERNT!

		next.ServeHTTP(w, r)
	})
}

// CSS-spezifische Headers
func addCSSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Richtiger MIME-Type für CSS
		w.Header().Set("Content-Type", "text/css; charset=utf-8")

		// Caching
		w.Header().Set("Cache-Control", "public, max-age=3600")

		next.ServeHTTP(w, r)
	})
}

func addStaticHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Type basierend auf Dateiendung setzen
		ext := strings.ToLower(filepath.Ext(r.URL.Path))

		switch ext {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".woff":
			w.Header().Set("Content-Type", "font/woff")
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		default:
			// Für unbekannte Dateien keinen Content-Type setzen
			// Der Go FileServer wird einen geeigneten setzen
		}

		// CORS für alle Static Files
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		// Caching für Static Files
		w.Header().Set("Cache-Control", "public, max-age=3600")

		// Security Headers nur für HTML/CSS (NICHT für JS!)
		if ext == ".html" || ext == ".css" {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		next.ServeHTTP(w, r)
	})
}

func serveJavaScript(w http.ResponseWriter, r *http.Request) {
	// File path extrahieren
	filePath := strings.TrimPrefix(r.URL.Path, "/js/")
	fullPath := filepath.Join("./static/js/", filePath)

	// Sicherheitscheck: Verhindere Directory Traversal
	if strings.Contains(filePath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Richtige Headers setzen
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Datei ausliefern
	http.ServeFile(w, r, fullPath)
}

// Test-Handler um Content-Type zu debuggen
func debugStaticHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"path":        r.URL.Path,
		"method":      r.Method,
		"headers":     r.Header,
		"user_agent":  r.UserAgent(),
		"remote_addr": r.RemoteAddr,
	}

	json.NewEncoder(w).Encode(response)
}

// Health Check Handler
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Uptime berechnen
	uptime := time.Since(startTime)

	// Memory Stats
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "comment-api",
		"version":   version,
		"stage":     stage,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    uptime.String(),
		"system": map[string]interface{}{
			"go_version":     runtime.Version(),
			"num_goroutines": runtime.NumGoroutine(),
			"memory_mb":      memStats.Alloc / 1024 / 1024,
			"gc_cycles":      memStats.NumGC,
		},
		"endpoints": map[string]string{
			"health": "/health",
			"api":    "/api/comments",
			"admin":  "/admin",
			"widget": "/js/comment-widget.js",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Erweiterte Health Check mit Redis-Status
func healthCheckWithRedisHandler(commentService *CommentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Redis-Verbindung testen
		redisStatus := "healthy"
		redisError := ""

		_, err := commentService.client.Ping(commentService.ctx).Result()
		if err != nil {
			redisStatus = "unhealthy"
			redisError = err.Error()
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Uptime berechnen
		uptime := time.Since(startTime)

		// Memory Stats
		var memStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStats)

		response := map[string]interface{}{
			"status":    "healthy",
			"service":   "comment-api",
			"version":   version,
			"stage":     stage,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"uptime":    uptime.String(),
			"dependencies": map[string]interface{}{
				"redis": map[string]interface{}{
					"status": redisStatus,
					"error":  redisError,
				},
			},
			"system": map[string]interface{}{
				"go_version":     runtime.Version(),
				"num_goroutines": runtime.NumGoroutine(),
				"memory_mb":      memStats.Alloc / 1024 / 1024,
				"gc_cycles":      memStats.NumGC,
				"platform":       runtime.GOOS + "/" + runtime.GOARCH,
			},
			"endpoints": map[string]string{
				"health": "/health",
				"api":    "/api/comments",
				"admin":  "/admin",
				"widget": "/js/comment-widget.js",
			},
		}

		json.NewEncoder(w).Encode(response)
	}
}

// Readiness Check (für Kubernetes)
func readinessCheckHandler(commentService *CommentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Redis-Verbindung testen
		_, err := commentService.client.Ping(commentService.ctx).Result()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			response := map[string]interface{}{
				"status": "not_ready",
				"error":  err.Error(),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"status": "ready",
		}
		json.NewEncoder(w).Encode(response)
	}
}

// Liveness Check (für Kubernetes)
func livenessCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "alive",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// Metrics Handler (für Monitoring)
func metricsHandler(commentService *CommentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Kommentar-Statistiken
		allComments, err := commentService.GetAllComments(true)
		if err != nil {
			http.Error(w, "Fehler beim Abrufen der Metriken", http.StatusInternalServerError)
			return
		}

		activeCount := 0
		inactiveCount := 0
		postIds := make(map[string]int)

		for _, comment := range allComments {
			if comment.Active {
				activeCount++
			} else {
				inactiveCount++
			}
			postIds[comment.PostID]++
		}

		// Memory Stats
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		response := map[string]interface{}{
			"metrics": map[string]interface{}{
				"comments_total":      len(allComments),
				"comments_active":     activeCount,
				"comments_inactive":   inactiveCount,
				"posts_with_comments": len(postIds),
				"uptime_seconds":      time.Since(startTime).Seconds(),
				"memory_bytes":        memStats.Alloc,
				"goroutines":          runtime.NumGoroutine(),
				"gc_cycles":           memStats.NumGC,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}
}

// Setup Health Check Routes
func setupHealthRoutes(r *mux.Router, commentService *CommentService) {
	// Standard Health Check
	r.HandleFunc("/health", healthCheckWithRedisHandler(commentService)).Methods("GET")
	r.HandleFunc("/", healthCheckHandler).Methods("GET") // Root auch für einfachen Check

	// Kubernetes Health Checks
	r.HandleFunc("/health/live", livenessCheckHandler).Methods("GET")
	r.HandleFunc("/health/ready", readinessCheckHandler(commentService)).Methods("GET")

	// Metrics (optional geschützt)
	r.HandleFunc("/metrics", metricsHandler(commentService)).Methods("GET")
}

// JavaScript Widget Handler mit Datei-Template
func (h *CommentHandler) JSWidgetHandler(w http.ResponseWriter, r *http.Request) {
	// Template laden (mit Caching)
	tmpl, err := loadJSTemplate()
	if err != nil {
		log.Printf("Template-Fehler: %v", err)
		http.Error(w, "Template nicht verfügbar", http.StatusInternalServerError)
		return
	}

	// API-URL dynamisch bestimmen
	apiUrl := determineApiUrl(r)

	// Template-Daten
	data := JSWidgetTemplateData{
		ApiUrl:  apiUrl,
		Version: version,
		Stage:   stage,
	}

	// Korrekte Headers für JavaScript
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800") // 30 Minuten Cache
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	// Template ausführen
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template-Ausführung fehlgeschlagen: %v", err)
		http.Error(w, "Template-Ausführung fehlgeschlagen", http.StatusInternalServerError)
		return
	}
}

// Template laden mit Caching und Hot-Reload im Development
func loadJSTemplate() (*template.Template, error) {
	templatePath := getEnv("JS_TEMPLATE_PATH", "./templates/comment-widget.js.tmpl")

	// Datei-Info abrufen
	info, err := os.Stat(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template-datei nicht gefunden: %s", templatePath)
	}

	jsTemplateCache.mutex.RLock()

	// Cache prüfen (nur in Produktion cachen)
	if stage == "production" && jsTemplateCache.template != nil &&
		jsTemplateCache.lastModified.Equal(info.ModTime()) {
		defer jsTemplateCache.mutex.RUnlock()
		return jsTemplateCache.template, nil
	}

	jsTemplateCache.mutex.RUnlock()

	// Template neu laden
	jsTemplateCache.mutex.Lock()
	defer jsTemplateCache.mutex.Unlock()

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template-parsing fehlgeschlagen: %w", err)
	}

	// Cache aktualisieren
	jsTemplateCache.template = tmpl
	jsTemplateCache.lastModified = info.ModTime()

	log.Printf("✅ JavaScript Template geladen: %s", templatePath)
	return tmpl, nil
}

// API-URL aus verschiedenen Quellen bestimmen
func determineApiUrl(r *http.Request) string {
	// Priorität der URL-Bestimmung:

	// 1. Environment Variable (höchste Priorität)
	if envUrl := getEnv("PUBLIC_API_URL", ""); envUrl != "" {
		return envUrl
	}

	// 2. Domain aus Environment + Schema detection
	if domain := getEnv("DOMAIN", ""); domain != "" {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s/api/comments", scheme, domain)
	}

	// 3. Aus Request-Host ableiten
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host
	return fmt.Sprintf("%s://%s/api/comments", scheme, host)
}

// Setup-Script für Template-Verzeichnis
func setupTemplateDirectory() error {
	templateDir := "./templates"

	// Verzeichnis erstellen falls nicht vorhanden
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("template-verzeichnis konnte nicht erstellt werden: %w", err)
	}

	templatePath := filepath.Join(templateDir, "comment-widget.js.tmpl")

	// Prüfen ob Template-Datei existiert
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Printf("⚠️  Template-Datei nicht gefunden: %s", templatePath)
		log.Println("💡 Bitte erstelle die Datei oder setze JS_TEMPLATE_PATH Environment Variable")
		return fmt.Errorf("template-datei fehlt: %s", templatePath)
	}

	log.Printf("✅ Template-Datei gefunden: %s", templatePath)
	return nil
}

// Template-Validator für Syntax-Prüfung
func validateJSTemplate() error {
	templatePath := getEnv("JS_TEMPLATE_PATH", "./templates/comment-widget.js.tmpl")

	// Template parsen (ohne Ausführung)
	_, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("template-syntax-fehler in %s: %w", templatePath, err)
	}

	// Test-Daten für Validation
	testData := JSWidgetTemplateData{
		ApiUrl:  "https://example.com/api/comments",
		Version: "1.0.0",
		Stage:   "test",
	}

	// Template mit Test-Daten ausführen (Dry-Run)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	// In Dummy-Writer ausführen
	err = tmpl.Execute(io.Discard, testData)
	if err != nil {
		return fmt.Errorf("template-ausführung fehlgeschlagen: %w", err)
	}

	log.Printf("✅ Template-Validation erfolgreich: %s", templatePath)
	return nil
}

// Development Helper: Template Hot-Reload
func enableTemplateHotReload() {
	if stage != "development" {
		return
	}

	templatePath := getEnv("JS_TEMPLATE_PATH", "./templates/comment-widget.js.tmpl")

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		var lastModTime time.Time

		for range ticker.C {
			info, err := os.Stat(templatePath)
			if err != nil {
				continue
			}

			if !lastModTime.Equal(info.ModTime()) {
				log.Printf("🔄 Template-Datei geändert, Cache wird geleert...")

				jsTemplateCache.mutex.Lock()
				jsTemplateCache.template = nil
				jsTemplateCache.lastModified = time.Time{}
				jsTemplateCache.mutex.Unlock()

				// Validation
				if err := validateJSTemplate(); err != nil {
					log.Printf("❌ Template-Validation fehlgeschlagen: %v", err)
				} else {
					log.Printf("✅ Template erfolgreich neu geladen")
				}

				lastModTime = info.ModTime()
			}
		}
	}()

	log.Println("🔥 Template Hot-Reload aktiviert (Development Mode)")
}

func main() {
	// Environment Variablen lesen
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvAsInt("REDIS_DB", 0)
	port := getEnv("PORT", "8080")

	log.Printf("🚀 Starting Comment API %s (%s)", version, stage)
	log.Printf("📡 Connecting to Redis: %s", redisAddr)

	// Auth-System initialisieren
	auth := NewTokenAuth()
	log.Printf("🔐 Authentication: %v", auth.Enabled)
	commentService := NewCommentService(redisAddr, redisPassword, redisDB)

	// Verbindung testen
	_, err := commentService.client.Ping(commentService.ctx).Result()
	if err != nil {
		log.Fatal("❌ Redis connection failed:", err)
	}
	log.Println("✅ Redis connection successful")

	// Template-Setup
	if err := setupTemplateDirectory(); err != nil {
		log.Fatal("Template-Setup fehlgeschlagen:", err)
	}

	// Template-Validation
	if err := validateJSTemplate(); err != nil {
		log.Fatal("Template-Validation fehlgeschlagen:", err)
	}

	// Hot-Reload im Development Mode
	enableTemplateHotReload()

	handler := NewCommentHandler(commentService)

	// Router einrichten
	r := mux.NewRouter()

	// Static Files ZUERST registrieren
	setupStaticRoutes(r)

	// JavaScript Widget Template
	r.HandleFunc("/js/comment-widget.js", handler.JSWidgetHandler).Methods("GET")

	// Health Check Routes ZUERST
	setupHealthRoutes(r, commentService)

	// Öffentliche Endpunkte
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")
	r.HandleFunc("/", healthCheckHandler).Methods("GET")

	// API-Endpunkte
	api := r.PathPrefix("/api/comments").Subrouter()
	api.HandleFunc("", handler.CreateCommentHandler).Methods("POST")
	api.HandleFunc("", handler.GetCommentsHandler).Methods("GET")
	api.HandleFunc("/{id}", handler.GetCommentHandler).Methods("GET")

	// Geschützte Admin-Endpunkte
	adminAPI := r.PathPrefix("/api/comments").Subrouter()
	adminAPI.Use(auth.AuthMiddleware) // Auth-Middleware anwenden
	adminAPI.HandleFunc("/{id}/status", handler.UpdateCommentStatusHandler).Methods("PUT")
	adminAPI.HandleFunc("/{id}", handler.DeleteCommentHandler).Methods("DELETE")
	adminAPI.HandleFunc("/admin/info", handler.AdminInfoHandler).Methods("GET")

	// Admin Panel (geschützt über HTTP Basic Auth oder Token in URL)
	adminPanel := r.PathPrefix("/admin").Subrouter()
	adminPanel.HandleFunc("/", handler.AdminPanelHandler).Methods("GET")
	adminPanel.HandleFunc("/panel", handler.AdminPanelHandler).Methods("GET")

	// CORS konfigurieren
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // In Produktion spezifischer konfigurieren
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	corsHandler := c.Handler(r)

	// Server Info
	fmt.Printf("🌐 Comment API %s running on port %s\n", version, port)
	fmt.Println("💚 Health Check Endpoints:")
	fmt.Println("  GET    /health                  - Detailed Health Check")
	fmt.Println("  GET    /health/live             - Liveness Check (K8s)")
	fmt.Println("  GET    /health/ready            - Readiness Check (K8s)")
	fmt.Println("  GET    /metrics                 - Application Metrics")
	fmt.Println("📋 API Endpoints:")
	fmt.Println("  GET    /                        - Simple Health Check")
	fmt.Println("  POST   /api/comments            - Create Comment")
	fmt.Println("  GET    /api/comments            - Get Comments")
	fmt.Println("📁 Static Files:")
	fmt.Println("  GET    /js/comment-widget.js    - Comment Widget")
	fmt.Println("🔐 Admin:")
	fmt.Println("  GET    /admin                   - Admin Panel")

	if auth.Enabled {
		fmt.Printf("🔑 Admin Token: %s\n", auth.AdminToken)
		fmt.Println("💡 Use: Authorization: Bearer <token> or X-Admin-Token: <token>")
	}
	log.Printf("📁 Template-Pfad: %s", getEnv("JS_TEMPLATE_PATH", "./templates/comment-widget.js.tmpl"))
	log.Printf("🎯 Template-Modus: %s", stage)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

// Template-Info Handler für Debugging
func (h *CommentHandler) TemplateInfoHandler(w http.ResponseWriter, r *http.Request) {
	templatePath := getEnv("JS_TEMPLATE_PATH", "./templates/comment-widget.js.tmpl")

	info, err := os.Stat(templatePath)
	if err != nil {
		http.Error(w, "Template-Info nicht verfügbar", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"template_path":   templatePath,
		"last_modified":   info.ModTime().Format(time.RFC3339),
		"size_bytes":      info.Size(),
		"cache_enabled":   stage == "production",
		"hot_reload":      stage == "development",
		"current_api_url": determineApiUrl(r),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
