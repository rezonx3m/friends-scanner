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
	"golang.org/x/crypto/bcrypt"
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

// hashPassword хеширует пароль с использованием bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPasswordHash проверяет, соответствует ли пароль хешу
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

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

	// Хешируем пароль перед сохранением
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		log.Printf("Ошибка хеширования пароля: %v", err)
		response := CreateEventResponse{Error: "Ошибка обработки пароля"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Создаем новый ивент с хешированным паролем
	_, err = db.Exec("INSERT INTO events (event_id, password) VALUES (?, ?)", req.EventID, hashedPassword)
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
	// Отключаем кеширование для HTML страницы с результатами
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	eventID := r.URL.Query().Get("event_id")
	password := r.URL.Query().Get("password")

	if eventID == "" {
		eventID = "default"
	}

	// Проверяем пароль для доступа к результатам
	var storedPasswordHash string
	err := db.QueryRow("SELECT password FROM events WHERE event_id = ?", eventID).Scan(&storedPasswordHash)
	if err == sql.ErrNoRows {
		http.Error(w, "Ивент не найден. Сначала создайте ивент.", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Ошибка получения пароля ивента: %v", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	// Проверяем пароль с использованием bcrypt
	if !checkPasswordHash(password, storedPasswordHash) {
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

	// Загружаем шаблон из файла
	t, err := template.ParseFiles("./static/results.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
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

	// Отключаем кеширование для HTML файлов
	if strings.HasSuffix(path, ".html") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	http.ServeFile(w, r, filePath)
}

func docHandler(w http.ResponseWriter, r *http.Request) {
	// Отключаем кеширование для страницы документации
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Обслуживание документации - отдаем doc.html
	http.ServeFile(w, r, "./static/doc.html")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Отключаем кеширование для главной страницы
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

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
