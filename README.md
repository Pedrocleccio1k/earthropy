# EARTHROPY

O "centro" do EARTHROPY. Este repositório **não tem o código dos apps**: ele diz
**quais apps existem, em qual endereço cada um abre e como tudo é instalado** no servidor.

```
            http://SERVIDOR:9090
                     │
              ┌──────┴──────┐
              │   Porteiro  │  (Caddy — caddy/Caddyfile)
              └──────┬──────┘
      ┌──────────────┼───────────────┬──────────────────────┐
      /           /drive/         /system/        /earthropy/apps.json
  Dashboard      A Drive           System          lista central de apps
 (Homarr)   (FileBrowser Quantum) (esta pasta)    (config/apps.json)
```

## Os repositórios

| Repositório | O que tem | Imagem gerada pelo GitHub |
|---|---|---|
| **earthropy** (este) | porteiro, lista de apps, instalador, app System | — (usa arquivos direto) |
| **earthropy-dashboard** | Dashboard (Homarr modificado) | `ghcr.io/<usuario>/earthropy-dashboard` |
| **earthropy-drive** | A Drive (FileBrowser Quantum modificado) | `ghcr.io/<usuario>/earthropy-drive` |

Cada app novo (Musik, Notz…) vai ganhar o seu próprio repositório e entrar aqui
com 3 linhas: uma em `config/apps.json`, um bloco no `caddy/Caddyfile` e um serviço
no `docker-compose.yml`.

## O que tem em cada pasta

| Caminho | Pra que serve |
|---|---|
| `config/apps.json` | **A lista de apps.** A barra do Dashboard, o menu de apps do A Drive e o System leem daqui. Colocou um endereço em `url` → o app aparece ativo em todo lugar. `null` → aparece como "ainda não instalado". |
| `caddy/Caddyfile` | O porteiro: qual caminho vai pra qual app. |
| `docker-compose.yml` | Todos os apps do servidor num arquivo só. |
| `install.sh` | Instalador: faz as perguntas, cria as configurações e liga tudo. |
| `config/drive/config.example.yaml` | Modelo de configuração do A Drive (o instalador cria a sua cópia). |
| `system/` | O app **System**: mostra os apps, se estão online, o mapa e como funcionam as atualizações. |
| `docs/arquitetura.md` | Explicação mais detalhada de como tudo se conecta. |

## Instalar no servidor

```bash
git clone https://github.com/<seu-usuario>/earthropy.git ~/earthropy
cd ~/earthropy
./install.sh
```

Depois abra `http://IP-DO-SERVIDOR:9090`.

## Atualizar

- **Apps** (Dashboard, A Drive): automático. O GitHub gera a imagem nova e o
  Watchtower troca no servidor em até 5 minutos.
- **Este repositório** (lista de apps, porteiro): no servidor, rode `./install.sh update`.

## O que nunca vai pro GitHub

`.env`, `data/` e `config/drive/config.yaml` ficam só no servidor (estão no
`.gitignore`). Senhas e arquivos ficam com você.
