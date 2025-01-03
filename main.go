package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Database struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		DBName   string `json:"dbname"`
	} `json:"database"`
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
}

var db *sql.DB
var config Config

// Mahasiswa struct
type Mahasiswa struct {
	NPM       int    `json:"npm"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Task struct
type Task struct {
	ID           int    `json:"id"`
	Text         string `json:"text"`
	Completed    bool   `json:"completed"`
	Deadline     string `json:"deadline"`
	MahasiswaNPM int    `json:"mahasiswa_npm"`
}

// Fungsi untuk memuat konfigurasi
func loadConfig() {
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal("Error reading config.json: ", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatal("Error parsing config.json: ", err)
	}
}

// Fungsi untuk inisialisasi database
func initDB() {
	loadConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", 
		config.Database.Username, 
		config.Database.Password, 
		config.Database.Host, 
		config.Database.Port, 
		config.Database.DBName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	fmt.Println("Database connected successfully!")
}

// Fungsi untuk mengambil data task dari database
func getTasksFromDB() ([]Task, error) {
	rows, err := db.Query("SELECT id, text, completed, deadline, mahasiswa_npm FROM tasks")
	if err != nil {
		return nil, fmt.Errorf("Error querying database: %v", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var deadline sql.NullString
		err := rows.Scan(&task.ID, &task.Text, &task.Completed, &deadline, &task.MahasiswaNPM)
		if err != nil {
			return nil, fmt.Errorf("Error scanning row: %v", err)
		}

		if deadline.Valid {
			task.Deadline = deadline.String
		} else {
			task.Deadline = ""
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating over rows: %v", err)
	}

	return tasks, nil
}

// Fungsi untuk menyimpan task ke database
func saveTaskToDB(task Task) error {
	_, err := db.Exec("INSERT INTO tasks (text, completed, deadline, mahasiswa_npm) VALUES (?, ?, ?, ?)", 
		task.Text, task.Completed, task.Deadline, task.MahasiswaNPM)
	return err
}

// Fungsi untuk menghapus task dari database
func deleteTaskFromDB(id int) error {
	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

// Handler untuk halaman utama
func serveHome(w http.ResponseWriter, r *http.Request) {
	var taskCount int
	err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
	if err != nil {
		http.Error(w, "Error checking tasks", http.StatusInternalServerError)
		return
	}

	if taskCount == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tasks, err := getTasksFromDB()
	if err != nil {
		http.Error(w, "Error fetching tasks", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, tasks)
}

// Handler untuk login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var loginMahasiswa Mahasiswa
		err := json.NewDecoder(r.Body).Decode(&loginMahasiswa)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		query := "SELECT npm, password FROM mahasiswa WHERE npm = ?"
		row := db.QueryRow(query, loginMahasiswa.NPM)

		var storedPassword string
		var mahasiswaNPM int
		err = row.Scan(&mahasiswaNPM, &storedPassword)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(loginMahasiswa.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Simpan session
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Handler untuk register
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		npm := r.FormValue("npm")
		username := r.FormValue("username")
		password := r.FormValue("password")

		if npm == "" || username == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("INSERT INTO mahasiswa (npm, username, password) VALUES (?, ?, ?)", npm, username, string(hashedPassword))
		if err != nil {
			http.Error(w, "Failed to register", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		http.ServeFile(w, r, "templates/register.html")
	}
}

func main() {
	initDB()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
