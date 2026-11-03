# EARTHROPY — guia para quem (ou o que) mexe no código

Servidor pessoal em **um executável Go**, sem Docker, sem dependências de sistema. A interface inteira é `ui/index.html` (HTML+CSS+JS puro, embutido no binário).

## Regras do projeto
- Mínimo de código, máximo de desempenho: biblioteca padrão primeiro; só três dependências externas (`modernc.org/sqlite`, `x/crypto/bcrypt`, `minio/selfupdate`). Não adicionar frameworks.
- Nada de CGO: o binário precisa cross-compilar para Linux/Windows/macOS no GitHub Actions.
- Textos da interface e mensagens de erro em português do Brasil.
- O servidor não precisa entender o estado da interface: ele é um JSON opaco em `kv.state` (compartilhado) + `prefs` (por pessoa). Só `settings.{name,desc,mode,code,accessCode,approve}` e `people[]` são lidos no Go (`stateView`).
- Arquivos ficam no disco de verdade em `<dados>/files`; pastas direcionadas a apps ficam em `kv.apps` (caminho → app). Lixeira em `<dados>/trash` + tabela `trash`.
- Build local: `GOSUMDB=off go build -ldflags "-s -w" .` (o sandbox bloqueia sum.golang.org; na máquina do Pedro não precisa de `GOSUMDB`).
- Testar: rode `./earthropy -porta 7799 -dados /tmp/epdata -sem-navegador` e use curl ou Playwright contra `http://localhost:7799`. `window.EP` expõe `DB`, `openApp`, `drive`, `whatch`, `ajustes` para testes.

## Layout
- `main.go` flags, pastas, abre o navegador · `db.go` SQLite/usuários/sessões · `server.go` rotas, auth, estado · `fs.go` arquivos, upload em fluxo, Range · `update.go` auto-atualização via `latest.json` da última Release.
- `.github/workflows/release.yml`: tag `vX.Y.Z` → binários + `latest.json`.
