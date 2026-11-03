package main

import (
	"crypto"
	_ "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/minio/selfupdate"
)

// Repo é "dono/repositorio" no GitHub; preenchido pelo build.
var Repo = "earthropy/earthropy"

type manifest struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	Assets  map[string]struct {
		URL    string `json:"url"`
		Sha256 string `json:"sha256"`
	} `json:"assets"`
}

func fetchManifest() (*manifest, error) {
	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Get("https://github.com/" + Repo + "/releases/latest/download/latest.json")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("sem releases publicadas ainda")
	}
	var m manifest
	return &m, json.NewDecoder(res.Body).Decode(&m)
}

func (s *server) updateCheck(w http.ResponseWriter, r *http.Request) {
	m, err := fetchManifest()
	if err != nil {
		jsonOut(w, map[string]any{"current": Version, "error": err.Error()})
		return
	}
	_, ok := m.Assets[runtime.GOOS+"-"+runtime.GOARCH]
	jsonOut(w, map[string]any{"current": Version, "latest": m.Version, "notes": m.Notes,
		"available": ok && m.Version != Version})
}

// updateApply baixa o binário do sistema atual, confere o SHA-256 do manifesto,
// troca o executável em uso e reinicia o processo com os mesmos argumentos.
func (s *server) updateApply(w http.ResponseWriter, r *http.Request) {
	m, err := fetchManifest()
	if err != nil {
		fail(w, 502, err.Error())
		return
	}
	a, ok := m.Assets[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		fail(w, 404, "sem versão para "+runtime.GOOS+"/"+runtime.GOARCH)
		return
	}
	res, err := http.Get(a.URL)
	if err != nil {
		fail(w, 502, err.Error())
		return
	}
	defer res.Body.Close()
	sum, err := hex.DecodeString(a.Sha256)
	if err != nil {
		fail(w, 502, "manifesto sem sha256")
		return
	}
	if err := selfupdate.Apply(res.Body, selfupdate.Options{Checksum: sum, Hash: crypto.SHA256}); err != nil {
		if rbErr := selfupdate.RollbackError(err); rbErr != nil {
			fail(w, 500, "atualização falhou e não deu para voltar: "+rbErr.Error())
			return
		}
		fail(w, 500, err.Error())
		return
	}
	jsonOut(w, map[string]string{"restarting": m.Version})
	go func() {
		time.Sleep(500 * time.Millisecond)
		exe, _ := os.Executable()
		c := exec.Command(exe, os.Args[1:]...)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if c.Start() == nil {
			os.Exit(0)
		}
	}()
}
