package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// Struct untuk User dan Task
type User struct {
	NPM       int    `json:"npm"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Task struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
	Deadline  string `json:"deadline"` // Format: dd/mm/yyyy (jam)
	MahasiswaNPM    int    `json:"mahasiswa_npm"`
}

var db *sql.DB

// Menyiapkan koneksi ke MySQL
func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/tasktracker") // Ganti username, password, dan dbname sesuai konfigurasi MySQL Anda
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
}

// Fungsi utama server
func main() {
	initDB() // Inisialisasi koneksi database

	// Routes
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/tasks", handleTasks)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Menjalankan server
	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// Menyajikan halaman home
func serveHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	// Mengambil daftar task dari database
	tasks, err := getTasksFromDB()
	if err != nil {
		http.Error(w, "Error fetching tasks", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, tasks)
}

// Menangani task (CRUD)
func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Ambil daftar task
		tasks, err := getTasksFromDB()
		if err != nil {
			http.Error(w, "Error fetching tasks", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	case "POST":
		var newTask Task
		err := json.NewDecoder(r.Body).Decode(&newTask)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Menyimpan task ke database
		err = saveTaskToDB(newTask)
		if err != nil {
			http.Error(w, "Error saving task", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case "DELETE":
		// Hapus semua task
		err := deleteAllTasks()
		if err != nil {
			http.Error(w, "Error deleting tasks", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Menyimpan task ke dalam database
func saveTaskToDB(task Task) error {
	_, err := db.Exec("INSERT INTO tasks (text, completed, deadline, mahasiswa_npm) VALUES (?, ?, ?, ?)",
		task.Text, task.Completed, task.Deadline, task.MahasiswaNPM)
	return err
}

// Mengambil daftar task dari database
func getTasksFromDB() ([]Task, error) {
	rows, err := db.Query("SELECT id, text, completed, deadline, mahasiswa_npm FROM tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Text, &task.Completed, &task.Deadline, &task.MahasiswaNPM)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Menghapus semua task dari database
func deleteAllTasks() error {
	_, err := db.Exec("DELETE FROM tasks")
	return err
}

// Fungsi Register
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var newUser User
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		newUser.Password = string(hashedPassword)

		// Menyimpan user ke database
		_, err = db.Exec("INSERT INTO mahasiswa (username, password) VALUES (?, ?)", newUser.Username, newUser.Password)
		if err != nil {
			http.Error(w, "Error saving user", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// Fungsi Login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var loginUser User
		err := json.NewDecoder(r.Body).Decode(&loginUser)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Ambil data user berdasarkan username
		row := db.QueryRow("SELECT id, password FROM mahasiswa WHERE username = ?", loginUser.Username)
		var storedPassword string
		var mahasiswaNPM int
		err = row.Scan(&mahasiswaNPM, &storedPassword)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Verifikasi password
		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(loginUser.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Set session atau token login untuk user
		// Misalnya redirect ke halaman utama
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
