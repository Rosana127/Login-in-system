package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var templates *template.Template
var store *sessions.CookieStore
var db *sql.DB

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	templates = template.Must(template.ParseGlob("templates/*.html"))

	var err error
	db, err = sql.Open("sqlite", "auth.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	// session & CSRF keys: 从环境变量读取，开发时可用默认值（但生产必须改）
	sessionKey := []byte(getEnv("SESSION_KEY", "dev-session-key-please-change"))
	store = sessions.NewCookieStore(sessionKey)
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7, // 7 days
		// Secure: true, // 生产启用 https 时设为 true
		SameSite: http.SameSiteLaxMode,
	}

	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler).Methods(http.MethodGet)
	r.HandleFunc("/login", loginPostHandler).Methods(http.MethodPost)
	r.HandleFunc("/logout", logoutHandler).Methods(http.MethodGet)
	r.HandleFunc("/", authRequired(homeHandler)).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	csrfKey := []byte(getEnv("CSRF_KEY", "32-byte-csrf-key-please-change-123456"))
	log.Println("Listening on :8080")
	http.ListenAndServe(":8080", csrf.Protect(csrfKey, csrf.Secure(false))(r))
}

// initDB: 创建 users 表并写入一个示例用户（username: alice password: test123）
func initDB(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`)
	if err != nil {
		return err
	}

	var cnt int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "alice").Scan(&cnt)
	if err != nil {
		return err
	}
	if cnt == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
		_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES (?,?)", "alice", string(hash))
		if err != nil {
			return err
		}
	}
	return nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CSRF": csrf.TemplateField(r),
	}
	if err := templates.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	var id int
	var pwHash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &pwHash)
	if err != nil {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(password)) != nil {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	session, _ := store.Get(r, "session")
	session.Values["user_id"] = id
	session.Values["username"] = username
	_ = session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, _ := session.Values["username"].(string)
	data := map[string]interface{}{
		"Username": username,
	}
	if err := templates.ExecuteTemplate(w, "home.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["user_id"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
