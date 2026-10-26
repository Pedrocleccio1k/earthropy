# Como o EARTHROPY se conecta

## 1. Uma porta só

Tudo abre em `http://SERVIDOR:9090`. Quem recebe é o **porteiro** (Caddy), que olha
o começo do endereço e manda pro app certo:

| Endereço | Vai para | Container |
|---|---|---|
| `/` (e tudo que não for abaixo) | Dashboard | `earthropy-dashboard` (porta 7575 interna) |
| `/drive/` | A Drive | `earthropy-drive` (porta 80 interna) |
| `/system/` | System | arquivos da pasta `system/` |
| `/earthropy/apps.json` | lista de apps | arquivo `config/apps.json` |

Os apps não têm porta própria aberta pra fora: só o porteiro tem. Como tudo fica no
mesmo endereço, o login, os links e a lista de apps funcionam sem configuração extra.

O A Drive sabe que mora em `/drive/` por causa da linha `baseURL: "/drive/"` na
configuração dele.

## 2. A lista central de apps

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

## 3. Como entra um app novo (exemplo: Musik)

1. Criar o repositório `earthropy-musik` com o workflow que gera a imagem.
2. Neste repositório:
   - `docker-compose.yml`: adicionar o serviço `musik` com o label do Watchtower.
   - `caddy/Caddyfile`: adicionar `handle /musik/* { reverse_proxy musik:PORTA }`.
   - `config/apps.json`: trocar `"url": null` por `"url": "/musik/"`.
3. No servidor: `./install.sh update`.

## 4. Atualizações

```
Você muda algo  →  push no GitHub  →  GitHub Actions gera a imagem (10–20 min)
       →  Watchtower vê a imagem nova (a cada 5 min)  →  reinicia só aquele app
```

Os dados ficam fora dos containers (`data/` e a sua pasta de arquivos), então
atualizar um app nunca apaga nada.

## 5. Onde ficam os dados no servidor

| Pasta | Conteúdo |
|---|---|
| `data/dashboard/` | banco do Dashboard (quadros, usuários, apps) |
| `data/drive/` | banco do A Drive (usuários, compartilhamentos) |
| `config/drive/config.yaml` | configurações do A Drive (com a sua senha de admin) |
| pasta escolhida no instalador (ex.: `/DATA`) | os seus arquivos de verdade |
