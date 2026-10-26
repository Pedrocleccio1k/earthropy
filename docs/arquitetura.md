# Como o EARTHROPY se conecta

## 1. Uma porta só

Tudo abre em `http://SERVIDOR:9090`. Quem recebe é o **porteiro** (Caddy), que olha
o começo do endereço e manda pro app certo:

| Endereço | Vai para | Container |
|---|---|---|
| `/` (e tudo que não for abaixo) | Dashboard | `earthropy-dashboard` (porta 7575 interna) |
| `/drive/` | A Drive | `earthropy-drive` (porta 80 interna) |
| `/drive/public/…` | links públicos de compartilhamento do A Drive (abrem sem login) | `earthropy-drive` |
| `/whatch/` | Whatch (Jellyfin) | `earthropy-whatch` (porta 8096 interna) |
| `/system/` | System | arquivos da pasta `system/` |
| `/earthropy/apps.json` | lista de apps | arquivo `config/apps.json` |

Os apps não têm porta própria aberta pra fora: só o porteiro tem. Como tudo fica no
mesmo endereço, o login, os links e a lista de apps funcionam sem configuração extra.

O A Drive sabe que mora em `/drive/` por causa da linha `baseURL: "/drive/"` na
configuração dele.

## 2. Login único

Você entra **uma vez**, na tela de login do Dashboard. A sessão fica guardada no navegador
até você sair.

```
navegador → /drive/...  →  porteiro: "quem é?"  →  Dashboard (/api/earthropy/auth)
                                │                         │
                                │      logado: "pedro, admin"     não logado: vai pro login
                                ▼                                  e depois volta pra cá
                         A Drive recebe o cabeçalho
                         X-Earthropy-User: pedro  →  entra direto como pedro
```

- **Quem é admin no Dashboard é admin no A Drive.** As outras pessoas entram como usuário comum.
- **Sair** (no Dashboard ou no botão de sair do A Drive) encerra a sessão do Dashboard, e
  com isso todos os apps pedem login de novo.
- O porteiro **apaga** qualquer cabeçalho `X-Earthropy-User` que venha do navegador, então
  ninguém consegue se passar por outra pessoa. Os apps também não têm porta aberta pra fora.
- O A Drive não tem mais senha própria. Uma conta antiga do A Drive com o mesmo nome de um
  usuário do Dashboard passa a ser dessa pessoa (com os arquivos e compartilhamentos dela).
- **Whatch (Jellyfin)** ainda tem login próprio, porque os apps do Jellyfin pro celular e pra TV
  falam direto com ele. Ligar ele no login único é um passo futuro.

## 3. A lista central de apps

`config/apps.json` é a única fonte de verdade:

```json
{ "id": "drive", "name": "A Drive", "letter": "A",
  "from": "#1b6bff", "to": "#19d3c5", "url": "/drive/" }
```

Quem lê essa lista:

- **Barra de apps do Dashboard**: mostra os apps; os que têm `url` ficam clicáveis.
  Se a lista não puder ser lida (ex.: Dashboard rodando sozinho), ela ainda usa os apps
  cadastrados manualmente no Homarr com o mesmo nome.
- **Menu de apps do A Drive**: clicar em "A Drive" no topo da barra lateral abre um
  painel com todos os apps, incluindo o Dashboard.
- **System**: mostra status, mapa e o mesmo menu.

## 4. Como entra um app novo (exemplo: Musik)

1. Criar o repositório `earthropy-musik` com o workflow que gera a imagem.
2. Neste repositório:
   - `docker-compose.yml`: adicionar o serviço `musik` com o label do Watchtower.
   - `caddy/Caddyfile`: adicionar `handle /musik/* { reverse_proxy musik:PORTA }`.
   - `config/apps.json`: trocar `"url": null` por `"url": "/musik/"`.
3. No servidor: `./install.sh update`.

## 5. Atualizações

```
Você muda algo  →  push no GitHub  →  GitHub Actions gera a imagem (10–20 min)
       →  Watchtower vê a imagem nova (a cada 5 min)  →  reinicia só aquele app
```

Os dados ficam fora dos containers (`data/` e a sua pasta de arquivos), então
atualizar um app nunca apaga nada.

## 6. Onde ficam os dados no servidor

| Pasta | Conteúdo |
|---|---|
| `data/dashboard/` | banco do Dashboard (quadros, usuários, apps) |
| `data/drive/` | banco do A Drive (usuários, compartilhamentos) |
| `data/whatch/` | configurações e biblioteca do Whatch (Jellyfin) |
| pasta escolhida no instalador (ex.: `/DATA`) | os seus arquivos de verdade |
