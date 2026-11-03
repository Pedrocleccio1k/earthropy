package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type server struct {
	db                *sql.DB
	root, trash, data string
}

type ctxKey struct{}

func (s *server) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, ui, "ui/index.html")
	})
	// público
	m.HandleFunc("GET /api/info", s.info)
	m.HandleFunc("POST /api/setup", s.setup)
	m.HandleFunc("POST /api/signup", s.signup)
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.logout)
	// autenticado
	a := s.auth
	m.HandleFunc("GET /api/state", a(s.getState))
	m.HandleFunc("PUT /api/state", a(s.putState))
	m.HandleFunc("POST /api/state", a(s.putState)) // sendBeacon só faz POST
	m.HandleFunc("POST /api/password", a(s.password))
	m.HandleFunc("GET /api/fs", a(s.fsAll))
	m.HandleFunc("POST /api/fs/{op}", a(s.fsOp))
	m.HandleFunc("POST /api/upload", a(s.upload))
	m.HandleFunc("GET /files/{p...}", a(s.serveFile))
	m.HandleFunc("POST /api/wallpaper", a(s.wallpaper))
	m.HandleFunc("GET /wallpapers/{p...}", a(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(s.data, "wallpapers", filepath.Base(r.PathValue("p"))))
	}))
	m.HandleFunc("GET /api/update", a(s.updateCheck))
	m.HandleFunc("POST /api/update", a(s.updateApply))
	return m
}

// ---------- utilidades ----------

func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) { http.Error(w, msg, code) }

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(v)
}

// safe transforma um caminho relativo vindo do cliente em um caminho absoluto
// dentro de root, sem jamais sair dele.
func (s *server) safe(rel string) (string, error) {
	p := filepath.Join(s.root, filepath.FromSlash(path.Clean("/"+rel)))
	if p != s.root && !strings.HasPrefix(p, s.root+string(filepath.Separator)) {
		return "", errors.New("caminho inválido")
	}
	return p, nil
}

func (s *server) user(r *http.Request) string {
	if u, ok := r.Context().Value(ctxKey{}).(string); ok {
		return u
	}
	return ""
}

// ---------- autenticação ----------

func (s *server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("ep")
		if err != nil {
			fail(w, 401, "faça login")
			return
		}
		u := sessionUser(s.db, c.Value)
		if u == "" {
			fail(w, 401, "sessão expirada")
			return
		}
		h(w, r.WithContext(withUser(r, u)))
	}
}

func (s *server) info(w http.ResponseWriter, r *http.Request) {
	st := s.stateView()
	jsonOut(w, map[string]any{"version": Version, "setup": userCount(s.db) == 0,
		"name": st.Settings.Name, "desc": st.Settings.Desc, "mode": st.Settings.Mode,
		"code": st.Settings.Code, "approve": st.Settings.Approve, "people": st.publicPeople()})
}

func (s *server) setup(w http.ResponseWriter, r *http.Request) {
	if userCount(s.db) > 0 {
		fail(w, 403, "servidor já configurado")
		return
	}
	var in struct{ Name, Pass string }
	if readJSON(r, &in) != nil || len(in.Pass) < 8 || in.Name == "" {
		fail(w, 400, "nome e senha (8+) obrigatórios")
		return
	}
	if err := createUser(s.db, in.Name, in.Pass); err != nil {
		fail(w, 500, err.Error())
		return
	}
	s.startSession(w, in.Name)
	jsonOut(w, map[string]string{"ok": in.Name})
}

func (s *server) signup(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Pass, Code, Acct string }
	if readJSON(r, &in) != nil || len(in.Pass) < 8 || in.Name == "" {
		fail(w, 400, "nome e senha (8+) obrigatórios")
		return
	}
	st := s.stateView()
	if st.Settings.Mode == "fechado" {
		fail(w, 403, "servidor fechado")
		return
	}
	if st.Settings.Code && in.Code != st.Settings.AccessCode {
		fail(w, 403, "código de acesso errado")
		return
	}
	if err := createUser(s.db, in.Name, in.Pass); err != nil {
		fail(w, 409, "nome já em uso")
		return
	}
	approved := !st.Settings.Approve
	if p := st.person(in.Name); p != nil { // admin já tinha adicionado a pessoa
		approved = p.Approved
	} else {
		s.addPerson(in.Name, in.Acct, approved)
	}
	jsonOut(w, map[string]bool{"approved": approved})
}

// addPerson acrescenta a pessoa ao estado da interface sem o servidor precisar
// entender o resto do JSON: lê como mapa, acrescenta, grava.
func (s *server) addPerson(name, acct string, approved bool) {
	var st map[string]any
	if json.Unmarshal([]byte(kvGet(s.db, "state", "{}")), &st) != nil || st == nil {
		st = map[string]any{}
	}
	people, _ := st["people"].([]any)
	colors := []string{"#f06292,#ff8a65", "#22d3e6,#1e7dff", "#4caf7a,#2fae5f", "#ffd166,#f4a261", "#7c5cff,#b39cff"}
	apps, role := []string{"drive", "whatch"}, "membro"
	if acct == "Infantil" {
		apps, role = []string{"whatch"}, "infantil"
	} else if acct == "" {
		acct = "Aberta"
	}
	quota := 200.0
	if set, ok := st["settings"].(map[string]any); ok {
		if q, ok := set["quotaDefault"].(float64); ok {
			quota = q
		}
	}
	st["people"] = append(people, map[string]any{"id": time.Now().Format("060102150405"), "n": name, "role": role, "acct": acct,
		"quota": quota * 1073741824, "used": 0, "g": colors[len(people)%len(colors)], "online": false, "apps": apps, "approved": approved})
	b, _ := json.Marshal(st)
	kvSet(s.db, "state", string(b))
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Pass string }
	if readJSON(r, &in) != nil {
		fail(w, 400, "json inválido")
		return
	}
	name, ok := checkUser(s.db, in.Name, in.Pass)
	if !ok {
		time.Sleep(400 * time.Millisecond) // desacelera tentativas
		fail(w, 401, "usuário ou senha errados")
		return
	}
	in.Name = name
	if p := s.stateView().person(in.Name); p != nil && !p.Approved {
		fail(w, 403, "conta aguardando aprovação")
		return
	}
	s.startSession(w, in.Name)
	jsonOut(w, map[string]string{"ok": in.Name})
}

func (s *server) startSession(w http.ResponseWriter, user string) {
	t, _ := newSession(s.db, user)
	http.SetCookie(w, &http.Cookie{Name: "ep", Value: t, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 180 * 24 * 3600})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("ep"); err == nil {
		dropSession(s.db, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "ep", Path: "/", MaxAge: -1})
	w.WriteHeader(204)
}

func (s *server) password(w http.ResponseWriter, r *http.Request) {
	var in struct{ Old, New string }
	if readJSON(r, &in) != nil || len(in.New) < 8 {
		fail(w, 400, "nova senha curta")
		return
	}
	if _, ok := checkUser(s.db, s.user(r), in.Old); !ok {
		fail(w, 400, "senha atual errada")
		return
	}
	setPassword(s.db, s.user(r), in.New)
	w.WriteHeader(204)
}

// ---------- estado da interface ----------

func (s *server) getState(w http.ResponseWriter, r *http.Request) {
	var prefs string
	s.db.QueryRow(`SELECT v FROM prefs WHERE user=?`, s.user(r)).Scan(&prefs)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"version":`+jsonStr(Version)+`,"me":`+jsonStr(s.user(r))+`,"state":`+orNull(kvGet(s.db, "state", ""))+`,"prefs":`+orNull(prefs)+`}`)
}

func (s *server) putState(w http.ResponseWriter, r *http.Request) {
	var in struct {
		State, Prefs json.RawMessage
	}
	if readJSON(r, &in) != nil {
		fail(w, 400, "json inválido")
		return
	}
	if len(in.State) > 0 {
		kvSet(s.db, "state", string(in.State))
	}
	if len(in.Prefs) > 0 {
		s.db.Exec(`INSERT INTO prefs(user,v) VALUES(?,?) ON CONFLICT(user) DO UPDATE SET v=excluded.v`, s.user(r), string(in.Prefs))
	}
	w.WriteHeader(204)
}

func jsonStr(s string) string { b, _ := json.Marshal(s); return string(b) }
func orNull(s string) string {
	if s == "" {
		return "null"
	}
	return s
}

// stateView lê só o que o servidor precisa entender do estado da interface.
type stateView struct {
	Settings struct {
		Name, Desc, Mode, AccessCode string
		Code, Approve                bool
	} `json:"settings"`
	People []person `json:"people"`
}
type person struct {
	N, Role, G string
	Approved   bool `json:"approved"`
}

func (s *server) stateView() (v stateView) {
	json.Unmarshal([]byte(kvGet(s.db, "state", "{}")), &v)
	return
}
func (v stateView) person(name string) *person {
	for i := range v.People {
		if strings.EqualFold(v.People[i].N, name) {
			return &v.People[i]
		}
	}
	return nil
}
func (v stateView) publicPeople() (out []map[string]string) {
	for _, p := range v.People {
		if p.Approved {
			out = append(out, map[string]string{"n": p.N, "g": p.G})
		}
	}
	return
}

// ---------- papéis de parede (fora da cota) ----------

func (s *server) wallpaper(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	f, h, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "arquivo ausente")
		return
	}
	defer f.Close()
	name := time.Now().Format("20060102-150405") + "-" + filepath.Base(h.Filename)
	out, err := os.Create(filepath.Join(s.data, "wallpapers", name))
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, f); err != nil {
		fail(w, 500, err.Error())
		return
	}
	jsonOut(w, map[string]string{"url": "/wallpapers/" + name})
}

func logErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
