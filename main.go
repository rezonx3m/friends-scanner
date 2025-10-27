package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ScannerRequest struct {
	EventID     string `json:"event_id"`
	UserID      string `json:"user_id"`
	ManagerName string `json:"manager_name"`
}

type ScannerResponse struct {
	Message string `json:"message"`
}

type CreateEventRequest struct {
	EventID  string `json:"event_id"`
	Password string `json:"password"`
}

type CreateEventResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type ScannerResult struct {
	Date        string `json:"date"`
	UserID      string `json:"user_id"`
	ManagerName string `json:"manager_name"`
	UserName    string `json:"user_name"`
	Avatar      string `json:"avatar"`
	ShortUserID string `json:"short_user_id"`
}

type PublicUser struct {
	Avatar           string `json:"avatar"`
	AvatarPreviewTpl string `json:"avatar_preview_tpl"`
	CreatedAt        int64  `json:"created_at"`
	FirstName        string `json:"first_name"`
	IsDeleted        bool   `json:"is_deleted"`
	LastName         string `json:"last_name"`
	Locale           string `json:"locale"`
	Name             string `json:"name"`
	Privacy          string `json:"privacy"`
	PublicUserID     string `json:"public_user_id"`
	UserID           int64  `json:"user_id"`
	UserIDStr        string `json:"user_id_str"`
}

type PublicProfileResponse struct {
	PublicUser PublicUser `json:"public_user"`
}

type ManagerStats struct {
	Name  string
	Count int
}

type Winner struct {
	UserID      string
	UserName    string
	Avatar      string
	ShortUserID string
	WonAt       string
}

type ResultsPageData struct {
	EventID      string
	Results      []ScannerResult
	TotalCount   int
	ManagerStats []ManagerStats
	Winners      []Winner
}

var db *sql.DB

func fetchUserProfile(userID string) (*PublicUser, error) {
	url := fmt.Sprintf("https://api.auth.2gis.com/public-profile/user/%s", userID)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API вернул статус: %d", resp.StatusCode)
	}

	var profileResp PublicProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profileResp); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %v", err)
	}

	return &profileResp.PublicUser, nil
}

func saveUserInfo(userID string, profile *PublicUser) error {
	// Формируем URL аватара с размерами 100x100
	avatarURL := profile.AvatarPreviewTpl
	if avatarURL != "" {
		avatarURL = strings.ReplaceAll(avatarURL, "{width}", "100")
		avatarURL = strings.ReplaceAll(avatarURL, "{height}", "100")
	}

	_, err := db.Exec(
		"INSERT OR REPLACE INTO users (user_id, short_user_id, name, avatar) VALUES (?, ?, ?, ?)",
		userID, profile.UserIDStr, profile.Name, avatarURL,
	)
	return err
}

func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", "./db.sqlite")
	if err != nil {
		return err
	}

	// Проверяем, существует ли таблица
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='friends_scanner';").Scan(&tableName)

	if err == sql.ErrNoRows {
		// Таблица не существует, создаем её
		createTableSQL := `
		CREATE TABLE friends_scanner (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			manager_name TEXT,
			add_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, user_id)
		);`

		_, err = db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("ошибка создания таблицы: %v", err)
		}
		log.Println("Таблица friends_scanner создана")
	} else if err != nil {
		return fmt.Errorf("ошибка проверки таблицы: %v", err)
	} else {
		log.Println("Таблица friends_scanner уже существует")
	}

	// Проверяем, существует ли таблица users
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users';").Scan(&tableName)

	if err == sql.ErrNoRows {
		// Таблица не существует, создаем её
		createUsersTableSQL := `
		CREATE TABLE users (
			user_id TEXT PRIMARY KEY,
			short_user_id TEXT,
			name TEXT,
			avatar TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`

		_, err = db.Exec(createUsersTableSQL)
		if err != nil {
			return fmt.Errorf("ошибка создания таблицы users: %v", err)
		}
		log.Println("Таблица users создана")
	} else if err != nil {
		return fmt.Errorf("ошибка проверки таблицы users: %v", err)
	} else {
		log.Println("Таблица users уже существует")
	}

	// Проверяем, существует ли таблица winners
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='winners';").Scan(&tableName)

	if err == sql.ErrNoRows {
		// Таблица не существует, создаем её
		createWinnersTableSQL := `
		CREATE TABLE winners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			won_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, user_id)
		);`

		_, err = db.Exec(createWinnersTableSQL)
		if err != nil {
			return fmt.Errorf("ошибка создания таблицы winners: %v", err)
		}
		log.Println("Таблица winners создана")
	} else if err != nil {
		return fmt.Errorf("ошибка проверки таблицы winners: %v", err)
	} else {
		log.Println("Таблица winners уже существует")
	}

	// Проверяем, существует ли таблица events
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events';").Scan(&tableName)

	if err == sql.ErrNoRows {
		// Таблица не существует, создаем её
		createEventsTableSQL := `
		CREATE TABLE events (
			event_id TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`

		_, err = db.Exec(createEventsTableSQL)
		if err != nil {
			return fmt.Errorf("ошибка создания таблицы events: %v", err)
		}
		log.Println("Таблица events создана")
	} else if err != nil {
		return fmt.Errorf("ошибка проверки таблицы events: %v", err)
	} else {
		log.Println("Таблица events уже существует")
	}

	return nil
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	var req ScannerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ScannerResponse{Message: "Invalid JSON"}
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.EventID == "" || req.UserID == "" {
		response := ScannerResponse{Message: "incorrect params"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Проверяем, существует ли такой ивент
	var eventExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM events WHERE event_id = ?)", req.EventID).Scan(&eventExists)
	if err != nil {
		log.Printf("Ошибка проверки существования ивента: %v", err)
		response := ScannerResponse{Message: "database error"}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !eventExists {
		response := ScannerResponse{Message: "event not found"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Получаем информацию о пользователе из API 2GIS
	profile, err := fetchUserProfile(req.UserID)
	if err != nil {
		log.Printf("Ошибка получения профиля пользователя %s: %v", req.UserID, err)
		// Продолжаем работу даже если не удалось получить профиль
	} else {
		// Сохраняем информацию о пользователе
		if err := saveUserInfo(req.UserID, profile); err != nil {
			log.Printf("Ошибка сохранения информации о пользователе: %v", err)
		}
	}

	_, err = db.Exec(
		"INSERT INTO friends_scanner (event_id, user_id, manager_name) VALUES (?, ?, ?)",
		req.EventID, req.UserID, req.ManagerName,
	)

	if err != nil {
		var response ScannerResponse
		if err.Error() == "UNIQUE constraint failed: friends_scanner.event_id, friends_scanner.user_id" {
			response.Message = "duplicate key value violates unique constraint"
		} else {
			response.Message = err.Error()
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ScannerResponse{Message: "ok"}
	json.NewEncoder(w).Encode(response)
}

func createEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := CreateEventResponse{Error: "Invalid JSON"}
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.EventID == "" || req.Password == "" {
		response := CreateEventResponse{Error: "event_id и password обязательны"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Проверяем, не существует ли уже такой ивент
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM events WHERE event_id = ?)", req.EventID).Scan(&exists)
	if err != nil {
		log.Printf("Ошибка проверки существования ивента: %v", err)
		response := CreateEventResponse{Error: "database error"}
		json.NewEncoder(w).Encode(response)
		return
	}

	if exists {
		response := CreateEventResponse{Error: "Ивент с таким ID уже существует"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Создаем новый ивент
	_, err = db.Exec("INSERT INTO events (event_id, password) VALUES (?, ?)", req.EventID, req.Password)
	if err != nil {
		log.Printf("Ошибка создания ивента: %v", err)
		response := CreateEventResponse{Error: "Ошибка создания ивента"}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := CreateEventResponse{Message: "Ивент успешно создан"}
	json.NewEncoder(w).Encode(response)
}

func selectWinnersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		EventID string `json:"event_id"`
		Count   int    `json:"count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.EventID == "" || req.Count <= 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "incorrect params"})
		return
	}

	// Получаем всех участников события, которые еще не победители
	rows, err := db.Query(`
		SELECT DISTINCT fs.user_id 
		FROM friends_scanner fs
		WHERE fs.event_id = ? 
		AND fs.user_id NOT IN (
			SELECT user_id FROM winners WHERE event_id = ?
		)
	`, req.EventID, req.EventID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		participants = append(participants, userID)
	}

	if len(participants) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Нет доступных участников"})
		return
	}

	// Ограничиваем количество победителей количеством участников
	winnersCount := req.Count
	if winnersCount > len(participants) {
		winnersCount = len(participants)
	}

	// Перемешиваем участников и выбираем победителей
	rand.Shuffle(len(participants), func(i, j int) {
		participants[i], participants[j] = participants[j], participants[i]
	})

	// Добавляем победителей в базу
	for i := 0; i < winnersCount; i++ {
		_, err := db.Exec(
			"INSERT INTO winners (event_id, user_id) VALUES (?, ?)",
			req.EventID, participants[i],
		)
		if err != nil {
			log.Printf("Ошибка добавления победителя: %v", err)
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func resetWinnersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.EventID == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "incorrect params"})
		return
	}

	_, err := db.Exec("DELETE FROM winners WHERE event_id = ?", req.EventID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event_id")
	password := r.URL.Query().Get("password")

	if eventID == "" {
		eventID = "default"
	}

	// Проверяем пароль для доступа к результатам
	var storedPassword string
	err := db.QueryRow("SELECT password FROM events WHERE event_id = ?", eventID).Scan(&storedPassword)
	if err == sql.ErrNoRows {
		http.Error(w, "Ивент не найден. Сначала создайте ивент.", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Ошибка получения пароля ивента: %v", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if password != storedPassword {
		http.Error(w, "Неверный пароль для доступа к результатам", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT 
			datetime(fs.add_time, 'localtime') as date, 
			fs.user_id, 
			fs.manager_name,
			u.name,
			u.avatar,
			u.short_user_id
		FROM friends_scanner fs
		LEFT JOIN users u ON fs.user_id = u.user_id
		WHERE fs.event_id = ? 
		ORDER BY fs.add_time DESC
	`, eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []ScannerResult
	for rows.Next() {
		var result ScannerResult
		var managerName, userName, avatar, shortUserID sql.NullString
		err := rows.Scan(&result.Date, &result.UserID, &managerName, &userName, &avatar, &shortUserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if managerName.Valid {
			result.ManagerName = managerName.String
		}
		if userName.Valid {
			result.UserName = userName.String
		}
		if avatar.Valid {
			result.Avatar = avatar.String
		}
		if shortUserID.Valid {
			result.ShortUserID = shortUserID.String
		}
		results = append(results, result)
	}

	// Подсчитываем статистику по менеджерам
	managerCounts := make(map[string]int)
	for _, result := range results {
		managerName := result.ManagerName
		if managerName == "" {
			managerName = "Без менеджера"
		}
		managerCounts[managerName]++
	}

	// Преобразуем в слайс для шаблона
	var managerStats []ManagerStats
	for name, count := range managerCounts {
		managerStats = append(managerStats, ManagerStats{
			Name:  name,
			Count: count,
		})
	}

	// Получаем победителей для данного события
	winnerRows, err := db.Query(`
		SELECT w.user_id, u.name, u.avatar, u.short_user_id, datetime(w.won_at, 'localtime') as won_at
		FROM winners w
		LEFT JOIN users u ON w.user_id = u.user_id
		WHERE w.event_id = ?
		ORDER BY w.won_at DESC
	`, eventID)
	if err != nil {
		log.Printf("Ошибка получения победителей: %v", err)
	}
	defer winnerRows.Close()

	var winners []Winner
	for winnerRows.Next() {
		var winner Winner
		var userName, avatar, shortUserID sql.NullString
		err := winnerRows.Scan(&winner.UserID, &userName, &avatar, &shortUserID, &winner.WonAt)
		if err != nil {
			log.Printf("Ошибка сканирования победителя: %v", err)
			continue
		}
		if userName.Valid {
			winner.UserName = userName.String
		}
		if avatar.Valid {
			winner.Avatar = avatar.String
		}
		if shortUserID.Valid {
			winner.ShortUserID = shortUserID.String
		}
		winners = append(winners, winner)
	}

	// HTML шаблон для таблицы
	tmpl := `
	<!DOCTYPE html>
	<html lang="ru">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Результаты сканирования - {{.EventID}}</title>
		<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body {
				font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
				background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
				min-height: 100vh;
				padding: 2rem 1rem;
				color: #333;
			}
			.container {
				max-width: 1200px;
				margin: 0 auto;
			}
			.header {
				background: rgba(255, 255, 255, 0.95);
				backdrop-filter: blur(10px);
				border-radius: 20px;
				padding: 2rem;
				margin-bottom: 2rem;
				text-align: center;
				box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
			}
			.header h1 {
				color: #4c51bf;
				font-size: 2rem;
				font-weight: 700;
				margin-bottom: 0.5rem;
			}
			.header p {
				color: #6b7280;
				font-size: 1.125rem;
			}
			.stats {
				display: grid;
				grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
				gap: 1rem;
				margin-bottom: 2rem;
			}
			.stat-card {
				background: rgba(255, 255, 255, 0.95);
				backdrop-filter: blur(10px);
				border-radius: 16px;
				padding: 1.5rem;
				text-align: center;
				box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
			}
			.stat-number {
				font-size: 2rem;
				font-weight: 700;
				color: #4c51bf;
				margin-bottom: 0.5rem;
			}
			.stat-label {
				color: #6b7280;
				font-weight: 500;
			}
			.table-container {
				background: rgba(255, 255, 255, 0.95);
				backdrop-filter: blur(10px);
				border-radius: 20px;
				overflow: hidden;
				box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
			}
			.results-table {
				width: 100%;
				border-collapse: collapse;
			}
			.results-table th {
				background: linear-gradient(135deg, #4c51bf 0%, #667eea 100%);
				color: #fff;
				padding: 1.25rem 1rem;
				text-align: left;
				font-weight: 600;
				font-size: 0.875rem;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.results-table td {
				padding: 1rem;
				border-bottom: 1px solid #e5e7eb;
				font-size: 0.875rem;
				color: #374151;
			}
			.results-table tr:hover {
				background: rgba(76, 81, 191, 0.05);
			}
			.results-table tr:last-child td {
				border-bottom: none;
			}
			.user-id {
				font-family: 'Monaco', 'Menlo', monospace;
				background: #f3f4f6;
				padding: 0.25rem 0.5rem;
				border-radius: 6px;
				font-weight: 500;
			}
			.manager-name {
				font-weight: 500;
				color: #059669;
			}
			.date-time {
				color: #6b7280;
			}
			.empty-state {
				text-align: center;
				padding: 3rem;
				color: #6b7280;
			}
			.empty-state h3 {
				font-size: 1.25rem;
				margin-bottom: 0.5rem;
			}
			.manager-stats-container {
				margin-bottom: 2rem;
			}
			.manager-stats-title {
				color: #4c51bf;
				font-size: 1.5rem;
				font-weight: 600;
				margin-bottom: 1rem;
				text-align: center;
			}
			.manager-stats-table {
				width: 100%;
				border-collapse: collapse;
			}
			.manager-stats-table th {
				background: linear-gradient(135deg, #059669 0%, #10b981 100%);
				color: #fff;
				padding: 1rem;
				text-align: left;
				font-weight: 600;
				font-size: 0.875rem;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.manager-stats-table td {
				padding: 0.875rem 1rem;
				border-bottom: 1px solid #e5e7eb;
				font-size: 0.875rem;
				color: #374151;
			}
			.manager-stats-table tr:hover {
				background: rgba(16, 185, 129, 0.05);
			}
			.manager-stats-table tr:last-child td {
				border-bottom: none;
			}
			.manager-count {
				font-weight: 600;
				color: #059669;
				text-align: center;
			}
			.user-avatar {
				width: 50px;
				height: 50px;
				border-radius: 50%;
				object-fit: cover;
			}
			.user-link {
				color: #4c51bf;
				text-decoration: none;
				font-family: 'Monaco', 'Menlo', monospace;
				font-weight: 500;
			}
			.user-link:hover {
				text-decoration: underline;
			}
			.action-buttons {
				display: flex;
				gap: 1rem;
				margin-bottom: 2rem;
				justify-content: center;
			}
			.btn {
				padding: 0.75rem 1.5rem;
				border: none;
				border-radius: 10px;
				font-weight: 600;
				font-size: 1rem;
				cursor: pointer;
				transition: all 0.3s ease;
				box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
			}
			.btn-primary {
				background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
				color: white;
			}
			.btn-primary:hover {
				transform: translateY(-2px);
				box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
			}
			.btn-danger {
				background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
				color: white;
			}
			.btn-danger:hover {
				transform: translateY(-2px);
				box-shadow: 0 6px 20px rgba(220, 38, 38, 0.4);
			}
			.modal {
				display: none;
				position: fixed;
				z-index: 1000;
				left: 0;
				top: 0;
				width: 100%;
				height: 100%;
				background-color: rgba(0, 0, 0, 0.5);
				backdrop-filter: blur(5px);
			}
			.modal-content {
				background: white;
				margin: 15% auto;
				padding: 2rem;
				border-radius: 20px;
				width: 90%;
				max-width: 400px;
				box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
			}
			.modal-title {
				color: #4c51bf;
				font-size: 1.5rem;
				font-weight: 700;
				margin-bottom: 1rem;
				text-align: center;
			}
			.modal-input {
				width: 100%;
				padding: 0.75rem;
				border: 2px solid #e5e7eb;
				border-radius: 10px;
				font-size: 1rem;
				margin-bottom: 1rem;
				box-sizing: border-box;
			}
			.modal-input:focus {
				outline: none;
				border-color: #667eea;
			}
			.modal-buttons {
				display: flex;
				gap: 1rem;
				justify-content: center;
			}
			.winners-container {
				margin-bottom: 2rem;
			}
			.winners-title {
				color: #dc2626;
				font-size: 1.5rem;
				font-weight: 600;
				margin-bottom: 1rem;
				text-align: center;
			}
			.winners-table {
				width: 100%;
				border-collapse: collapse;
			}
			.winners-table th {
				background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
				color: #fff;
				padding: 1rem;
				text-align: left;
				font-weight: 600;
				font-size: 0.875rem;
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.winners-table td {
				padding: 0.875rem 1rem;
				border-bottom: 1px solid #e5e7eb;
				font-size: 0.875rem;
				color: #374151;
			}
			.winners-table tr:hover {
				background: rgba(220, 38, 38, 0.05);
			}
			.winners-table tr:last-child td {
				border-bottom: none;
			}
			@media (max-width: 768px) {
				body { padding: 1rem 0.5rem; }
				.header { padding: 1.5rem; }
				.header h1 { font-size: 1.5rem; }
				.header p { font-size: 1rem; }
				.results-table th,
				.results-table td { padding: 0.75rem 0.5rem; font-size: 0.75rem; }
				.manager-stats-table th,
				.manager-stats-table td { padding: 0.75rem 0.5rem; font-size: 0.75rem; }
				.manager-stats-title { font-size: 1.25rem; }
				.user-avatar { width: 40px; height: 40px; }
				.stat-card { padding: 1rem; }
				.stat-number { font-size: 1.5rem; }
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>Результаты сканирования</h1>
				<p>Событие: {{.EventID}}</p>
			</div>
			
			<div class="stats">
				<div class="stat-card">
					<div class="stat-number">{{.TotalCount}}</div>
					<div class="stat-label">Всего регистраций</div>
				</div>
			</div>

			{{if .ManagerStats}}
			<div class="manager-stats-container">
				<h2 class="manager-stats-title">Статистика по менеджерам</h2>
				<div class="table-container">
					<table class="manager-stats-table">
						<thead>
							<tr>
								<th>Менеджер</th>
								<th>Количество</th>
							</tr>
						</thead>
						<tbody>
							{{range .ManagerStats}}
							<tr>
								<td class="manager-name">{{.Name}}</td>
								<td class="manager-count">{{.Count}}</td>
							</tr>
							{{end}}
						</tbody>
					</table>
				</div>
			</div>
			{{end}}

			<!-- Кнопки управления победителями -->
			<div class="action-buttons">
				{{if .Winners}}
				<button class="btn btn-danger" onclick="resetWinners()">Обнулить результаты</button>
				{{else}}
				<button class="btn btn-primary" onclick="showWinnerModal()">Определить победителя</button>
				{{end}}
			</div>

			<!-- Таблица победителей -->
			{{if .Winners}}
			<div class="winners-container">
				<h2 class="winners-title">🏆 Победители</h2>
				<div class="table-container">
					<table class="winners-table">
						<thead>
							<tr>
								<th>Дата выигрыша</th>
								<th>Аватар</th>
								<th>Имя</th>
								<th>Short ID</th>
								<th>User ID</th>
							</tr>
						</thead>
						<tbody>
							{{range .Winners}}
							<tr>
								<td class="date-time">{{.WonAt}}</td>
								<td>
									{{if .Avatar}}
									<img src="{{.Avatar}}" alt="{{.UserName}}" class="user-avatar">
									{{else}}
									<div class="user-avatar" style="background: #e5e7eb;"></div>
									{{end}}
								</td>
								<td>{{if .UserName}}{{.UserName}}{{else}}-{{end}}</td>
								<td>{{if .ShortUserID}}{{.ShortUserID}}{{else}}-{{end}}</td>
								<td><a href="https://2gis.ru/user/{{.UserID}}" target="_blank" class="user-link">{{.UserID}}</a></td>
							</tr>
							{{end}}
						</tbody>
					</table>
				</div>
			</div>
			{{end}}

			<div class="table-container">
				{{if .Results}}
				<table class="results-table">
					<thead>
						<tr>
							<th>Дата и время</th>
							<th>Аватар</th>
							<th>Имя</th>
							<th>Short ID</th>
							<th>User ID</th>
							<th>Менеджер</th>
						</tr>
					</thead>
					<tbody>
						{{range .Results}}
						<tr>
							<td class="date-time">{{.Date}}</td>
							<td>
								{{if .Avatar}}
								<img src="{{.Avatar}}" alt="{{.UserName}}" class="user-avatar">
								{{else}}
								<div class="user-avatar" style="background: #e5e7eb;"></div>
								{{end}}
							</td>
							<td>{{if .UserName}}{{.UserName}}{{else}}-{{end}}</td>
							<td>{{if .ShortUserID}}{{.ShortUserID}}{{else}}-{{end}}</td>
							<td><a href="https://2gis.ru/user/{{.UserID}}" target="_blank" class="user-link">{{.UserID}}</a></td>
							<td class="manager-name">{{if .ManagerName}}{{.ManagerName}}{{else}}-{{end}}</td>
						</tr>
						{{end}}
					</tbody>
				</table>
				{{else}}
				<div class="empty-state">
					<h3>Пока нет регистраций</h3>
					<p>Начните сканировать QR-коды для регистрации участников</p>
				</div>
				{{end}}
			</div>
		</div>

		<!-- Модальное окно для выбора количества победителей -->
		<div id="winnerModal" class="modal">
			<div class="modal-content">
				<h2 class="modal-title">Определить победителей</h2>
				<input type="number" id="winnerCount" class="modal-input" placeholder="Количество победителей" min="1" value="1">
				<div class="modal-buttons">
					<button class="btn btn-primary" onclick="selectWinners()">Выбрать</button>
					<button class="btn btn-danger" onclick="closeWinnerModal()">Отмена</button>
				</div>
			</div>
		</div>

		<script>
			const eventID = "{{.EventID}}";

			function showWinnerModal() {
				document.getElementById('winnerModal').style.display = 'block';
			}

			function closeWinnerModal() {
				document.getElementById('winnerModal').style.display = 'none';
			}

			function selectWinners() {
				const count = parseInt(document.getElementById('winnerCount').value);
				if (!count || count <= 0) {
					alert('Пожалуйста, введите корректное количество победителей');
					return;
				}

				fetch('/select-winners', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						event_id: eventID,
						count: count
					})
				})
				.then(response => response.json())
				.then(data => {
					if (data.error) {
						alert('Ошибка: ' + data.error);
					} else {
						location.reload();
					}
				})
				.catch(error => {
					alert('Ошибка при выборе победителей: ' + error);
				});
			}

			function resetWinners() {
				if (!confirm('Вы уверены, что хотите обнулить результаты?')) {
					return;
				}

				fetch('/reset-winners', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						event_id: eventID
					})
				})
				.then(response => response.json())
				.then(data => {
					if (data.error) {
						alert('Ошибка: ' + data.error);
					} else {
						location.reload();
					}
				})
				.catch(error => {
					alert('Ошибка при обнулении результатов: ' + error);
				});
			}

			// Закрытие модального окна при клике вне его
			window.onclick = function(event) {
				const modal = document.getElementById('winnerModal');
				if (event.target == modal) {
					closeWinnerModal();
				}
			}
		</script>
	</body>
	</html>`

	t, err := template.New("results").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Подготавливаем данные для шаблона
	pageData := ResultsPageData{
		EventID:      eventID,
		Results:      results,
		TotalCount:   len(results),
		ManagerStats: managerStats,
		Winners:      winners,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.Execute(w, pageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	// Обслуживание статических файлов из папки /static/
	path := r.URL.Path
	// Убираем префикс /static из пути
	path = strings.TrimPrefix(path, "/static")
	// Если путь пустой или это корень static, отдаем index.html
	if path == "/" || path == "" {
		path = "/index.html"
	}

	// Проверяем, существует ли файл
	filePath := "./static" + path
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

func docHandler(w http.ResponseWriter, r *http.Request) {
	// Обслуживание документации - отдаем doc.html
	http.ServeFile(w, r, "./static/doc.html")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Обслуживание корневого пути - отдаем index.html
	http.ServeFile(w, r, "./static/index.html")
}

func main() {
	// Парсинг флагов командной строки
	port := flag.String("port", "8080", "Порт для запуска сервера")
	flag.Parse()

	// Инициализация базы данных
	if err := initDatabase(); err != nil {
		log.Fatal("Ошибка инициализации базы данных:", err)
	}
	defer db.Close()

	// Создаем папку static если её нет
	if _, err := os.Stat("./static"); os.IsNotExist(err) {
		os.Mkdir("./static", 0755)
	}

	// Маршруты
	http.HandleFunc("/scan", scanHandler)
	http.HandleFunc("/create-event", createEventHandler)
	http.HandleFunc("/results", resultsHandler)
	http.HandleFunc("/select-winners", selectWinnersHandler)
	http.HandleFunc("/reset-winners", resetWinnersHandler)
	http.HandleFunc("/doc", docHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/", rootHandler)

	log.Printf("Сервер запущен на http://localhost:%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
