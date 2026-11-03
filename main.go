// EARTHROPY — servidor pessoal em um só executável.
// Sobe a interface em http://localhost:<porta>, guarda tudo em uma pasta de dados
// e se atualiza sozinho a partir das releases do GitHub.
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Version é preenchida pelo build (-ldflags "-X main.Version=v0.1.0").
var Version = "dev"

//go:embed ui/index.html
var ui embed.FS

func main() {
	port := flag.Int("porta", 7777, "porta HTTP")
	data := flag.String("dados", defaultDataDir(), "pasta de dados (banco, arquivos, lixeira)")
	noBrowser := flag.Bool("sem-navegador", false, "não abrir o navegador ao iniciar")
	flag.Parse()

	for _, d := range []string{"files", "trash", "wallpapers"} {
		if err := os.MkdirAll(filepath.Join(*data, d), 0o755); err != nil {
			log.Fatal(err)
		}
	}
	db, err := openDB(filepath.Join(*data, "earthropy.db"))
	if err != nil {
		log.Fatal(err)
	}
	s := &server{db: db, root: filepath.Join(*data, "files"), trash: filepath.Join(*data, "trash"), data: *data}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("porta %d ocupada: %v", *port, err)
	}
	url := fmt.Sprintf("http://localhost:%d", *port)
	log.Printf("EARTHROPY %s · %s · dados em %s", Version, url, *data)
	if !*noBrowser {
		go func() { time.Sleep(300 * time.Millisecond); openBrowser(url) }()
	}
	srv := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.Serve(ln))
}

func defaultDataDir() string {
	if d := os.Getenv("EARTHROPY_DATA"); d != "" {
		return d
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base, _ = os.UserHomeDir()
	}
	if runtime.GOOS == "linux" { // segue a XDG: ~/.local/share
		if h, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(h, ".local", "share")
		}
	}
	return filepath.Join(base, "earthropy")
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		c = exec.Command("open", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	_ = c.Start()
}
