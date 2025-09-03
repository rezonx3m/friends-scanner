package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

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

type ScannerResult struct {
	Date        string `json:"date"`
	UserID      string `json:"user_id"`
	ManagerName string `json:"manager_name"`
}

var db *sql.DB

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

	return nil
}

func scannerPostDataHandler(w http.ResponseWriter, r *http.Request) {
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

	_, err := db.Exec(
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

func scannerResultsHandler(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		eventID = "default"
	}

	rows, err := db.Query(
		"SELECT datetime(add_time, 'localtime') as date, user_id, manager_name FROM friends_scanner WHERE event_id = ? ORDER BY add_time DESC",
		eventID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []ScannerResult
	for rows.Next() {
		var result ScannerResult
		var managerName sql.NullString
		err := rows.Scan(&result.Date, &result.UserID, &managerName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if managerName.Valid {
			result.ManagerName = managerName.String
		}
		results = append(results, result)
	}

	// HTML шаблон для таблицы
	tmpl := `
	<table style='border-collapse: collapse; border:1px solid #69899F;'>
		<tr>
			<th>Дата</th><th>Пользователь</th><th>Добавляющий менеджер</th>
		</tr>
		{{range .}}
		<tr>
			<td style='border:1px dotted #000000; padding:5px;'>{{.Date}}</td>
			<td style='border:1px dotted #000000; padding:5px;'>{{.UserID}}</td>
			<td style='border:1px dotted #000000; padding:5px;'>{{.ManagerName}}</td>
		</tr>
		{{end}}
	</table>`

	t, err := template.New("results").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = t.Execute(w, results)
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
	http.HandleFunc("/scannerPostData", scannerPostDataHandler)
	http.HandleFunc("/scannerResults", scannerResultsHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/", rootHandler)

	log.Printf("Сервер запущен на http://localhost:%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
