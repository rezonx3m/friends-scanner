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

type ManagerStats struct {
	Name  string
	Count int
}

type ResultsPageData struct {
	EventID       string
	Results       []ScannerResult
	TotalCount    int
	ManagerStats  []ManagerStats
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

func resultsHandler(w http.ResponseWriter, r *http.Request) {
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

			<div class="table-container">
				{{if .Results}}
				<table class="results-table">
					<thead>
						<tr>
							<th>Дата и время</th>
							<th>ID пользователя</th>
							<th>Менеджер</th>
						</tr>
					</thead>
					<tbody>
						{{range .Results}}
						<tr>
							<td class="date-time">{{.Date}}</td>
							<td><span class="user-id">{{.UserID}}</span></td>
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
	</body>
	</html>`

	t, err := template.New("results").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

	// Подготавливаем данные для шаблона
	pageData := ResultsPageData{
		EventID:      eventID,
		Results:      results,
		TotalCount:   len(results),
		ManagerStats: managerStats,
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
	http.HandleFunc("/results", resultsHandler)
	http.HandleFunc("/doc", docHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/", rootHandler)

	log.Printf("Сервер запущен на http://localhost:%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
