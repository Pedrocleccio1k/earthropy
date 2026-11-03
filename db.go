package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Banco pequeno: tudo que não é arquivo fica aqui. O "estado" da interface
// (pessoas, ajustes, pastas direcionadas…) é um JSON único em kv — para
// cinco amigos isso é mais rápido e mais simples do que dezenas de tabelas.
const schema = `
PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS kv(k TEXT PRIMARY KEY, v TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS users(name TEXT PRIMARY KEY, hash BLOB NOT NULL, created INT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(token TEXT PRIMARY KEY, user TEXT NOT NULL, created INT NOT NULL);
CREATE TABLE IF NOT EXISTS prefs(user TEXT PRIMARY KEY, v TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS trash(id INTEGER PRIMARY KEY, orig TEXT NOT NULL, stored TEXT NOT NULL, at INT NOT NULL);`

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: um escritor; evita "database is locked"
	_, err = db.Exec(schema)
	return db, err
}

func kvGet(db *sql.DB, k, def string) string {
	var v string
	if db.QueryRow(`SELECT v FROM kv WHERE k=?`, k).Scan(&v) != nil {
		return def
	}
	return v
}

func kvSet(db *sql.DB, k, v string) error {
	_, err := db.Exec(`INSERT INTO kv(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, k, v)
	return err
}

func userCount(db *sql.DB) (n int) { db.QueryRow(`SELECT count(*) FROM users`).Scan(&n); return }

func createUser(db *sql.DB, name, pass string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(pass), 12)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users(name,hash,created) VALUES(?,?,?)`, name, h, time.Now().Unix())
	return err
}

// checkUser confere a senha e devolve o nome como foi cadastrado (a busca ignora maiúsculas).
func checkUser(db *sql.DB, name, pass string) (string, bool) {
	var h []byte
	var real string
	if db.QueryRow(`SELECT name,hash FROM users WHERE name=? COLLATE NOCASE`, name).Scan(&real, &h) != nil {
		return "", false
	}
	return real, bcrypt.CompareHashAndPassword(h, []byte(pass)) == nil
}

func setPassword(db *sql.DB, name, pass string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(pass), 12)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE users SET hash=? WHERE name=? COLLATE NOCASE`, h, name)
	return err
}

func newSession(db *sql.DB, user string) (string, error) {
	b := make([]byte, 24)
	rand.Read(b)
	t := hex.EncodeToString(b)
	_, err := db.Exec(`INSERT INTO sessions(token,user,created) VALUES(?,?,?)`, t, user, time.Now().Unix())
	return t, err
}

func sessionUser(db *sql.DB, token string) (u string) {
	db.QueryRow(`SELECT user FROM sessions WHERE token=?`, token).Scan(&u)
	return
}

func dropSession(db *sql.DB, token string) { db.Exec(`DELETE FROM sessions WHERE token=?`, token) }
