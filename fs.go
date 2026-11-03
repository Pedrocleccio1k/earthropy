package main

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func withUser(r *http.Request, u string) context.Context {
	return context.WithValue(r.Context(), ctxKey{}, u)
}

// apps: mapa pasta → app direcionado ("whatch", "whatch-series", "musik"…), em kv.
func (s *server) apps() map[string]string {
	m := map[string]string{}
	json.Unmarshal([]byte(kvGet(s.db, "apps", "{}")), &m)
	return m
}
func (s *server) saveApps(m map[string]string) {
	b, _ := json.Marshal(m)
	kvSet(s.db, "apps", string(b))
}

type dirEntry struct {
	Path, Name, Parent, App string
	Created                 int64
}
type fileEntry struct {
	Path, Name, Parent string
	Size, Modified     int64
}

// fsAll devolve a árvore inteira de uma vez: para uma biblioteca doméstica isso é
// uma única passada no disco e evita dezenas de requisições.
func (s *server) fsAll(w http.ResponseWriter, r *http.Request) {
	apps := s.apps()
	dirs, files := []dirEntry{}, []fileEntry{}
	filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == s.root {
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(p, s.root+string(filepath.Separator)))
		parent := path.Dir(rel)
		if parent == "." {
			parent = ""
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, dirEntry{rel, d.Name(), parent, apps[rel], info.ModTime().Unix()})
		} else if !strings.HasPrefix(d.Name(), ".") {
			files = append(files, fileEntry{rel, d.Name(), parent, info.Size(), info.ModTime().UnixMilli()})
		}
		return nil
	})
	var trash []map[string]any
	rows, _ := s.db.Query(`SELECT id,orig,stored,at FROM trash ORDER BY at DESC`)
	for rows != nil && rows.Next() {
		var id, at int64
		var orig, stored string
		rows.Scan(&id, &orig, &stored, &at)
		size := diskSize(filepath.Join(s.trash, stored))
		trash = append(trash, map[string]any{"id": id, "orig": orig, "at": at, "size": size, "name": path.Base(orig)})
	}
	jsonOut(w, map[string]any{"dirs": dirs, "files": files, "trash": trash})
}

// fsOp concentra as operações de arquivos: um endpoint, um switch.
func (s *server) fsOp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path, Name, To, App string
		ID                  int64
	}
	if readJSON(r, &in) != nil {
		fail(w, 400, "json inválido")
		return
	}
	src, err := s.safe(in.Path)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	var e error
	switch r.PathValue("op") {
	case "mkdir":
		dst := filepath.Join(src, filepath.Base(in.Name))
		if e = os.Mkdir(dst, 0o755); e == nil && in.App != "" {
			a := s.apps()
			a[relOf(s.root, dst)] = in.App
			s.saveApps(a)
		}
	case "rename":
		dst := filepath.Join(filepath.Dir(src), filepath.Base(in.Name))
		e = os.Rename(src, dst)
		s.moveApps(relOf(s.root, src), relOf(s.root, dst))
	case "move":
		var dir string
		if dir, e = s.safe(in.To); e == nil {
			dst := filepath.Join(dir, filepath.Base(src))
			e = os.Rename(src, dst)
			s.moveApps(relOf(s.root, src), relOf(s.root, dst))
		}
	case "app":
		a := s.apps()
		if in.App == "" {
			delete(a, in.Path)
		} else {
			a[in.Path] = in.App
		}
		s.saveApps(a)
	case "trash":
		stored := strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + filepath.Base(src)
		if e = os.Rename(src, filepath.Join(s.trash, stored)); e == nil {
			_, e = s.db.Exec(`INSERT INTO trash(orig,stored,at) VALUES(?,?,?)`, relOf(s.root, src), stored, time.Now().UnixMilli())
		}
	case "restore", "purge":
		var orig, stored string
		if s.db.QueryRow(`SELECT orig,stored FROM trash WHERE id=?`, in.ID).Scan(&orig, &stored) != nil {
			fail(w, 404, "item não está na lixeira")
			return
		}
		if r.PathValue("op") == "restore" {
			dst, _ := s.safe(orig)
			os.MkdirAll(filepath.Dir(dst), 0o755)
			e = os.Rename(filepath.Join(s.trash, stored), dst)
		} else {
			e = os.RemoveAll(filepath.Join(s.trash, stored))
		}
		if e == nil {
			s.db.Exec(`DELETE FROM trash WHERE id=?`, in.ID)
		}
	case "empty":
		os.RemoveAll(s.trash)
		os.MkdirAll(s.trash, 0o755)
		s.db.Exec(`DELETE FROM trash`)
	default:
		fail(w, 404, "operação desconhecida")
		return
	}
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	w.WriteHeader(204)
}

// diskSize soma o tamanho de um arquivo ou de tudo dentro de uma pasta.
func diskSize(p string) (n int64) {
	filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if i, e := d.Info(); e == nil {
				n += i.Size()
			}
		}
		return nil
	})
	return
}

func relOf(root, p string) string {
	return filepath.ToSlash(strings.TrimPrefix(p, root+string(filepath.Separator)))
}

// moveApps acompanha pastas direcionadas quando são renomeadas ou movidas.
func (s *server) moveApps(from, to string) {
	a := s.apps()
	changed := false
	for k, v := range a {
		if k == from || strings.HasPrefix(k, from+"/") {
			delete(a, k)
			a[to+strings.TrimPrefix(k, from)] = v
			changed = true
		}
	}
	if changed {
		s.saveApps(a)
	}
}

// upload recebe um multipart em fluxo: cada parte vai direto para o disco,
// sem passar pela memória; o nome do arquivo pode trazer subpastas.
func (s *server) upload(w http.ResponseWriter, r *http.Request) {
	base, err := s.safe(r.URL.Query().Get("p"))
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		fail(w, 400, "esperava multipart")
		return
	}
	n := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		// part.FileName() corta as subpastas; lemos o nome cru para manter a árvore enviada.
		_, params, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		rel := path.Clean("/" + strings.ReplaceAll(params["filename"], "\\", "/"))
		if rel == "/" {
			continue
		}
		dst := filepath.Join(base, filepath.FromSlash(rel))
		if !strings.HasPrefix(dst, base) {
			fail(w, 400, "caminho inválido")
			return
		}
		os.MkdirAll(filepath.Dir(dst), 0o755)
		f, err := os.Create(dst)
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		_, err = io.Copy(f, part)
		f.Close()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		n++
	}
	jsonOut(w, map[string]int{"saved": n})
}

// serveFile entrega qualquer arquivo com suporte a Range: é isso que faz o
// vídeo pular de posição no player sem baixar tudo.
func (s *server) serveFile(w http.ResponseWriter, r *http.Request) {
	p, err := s.safe(r.PathValue("p"))
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	if r.URL.Query().Get("dl") != "" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(p)+"\"")
	}
	http.ServeFile(w, r, p)
}
