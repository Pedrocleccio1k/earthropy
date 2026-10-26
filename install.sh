#!/usr/bin/env bash
# EARTHROPY installer
#   ./install.sh          first install (asks a few questions)
#   ./install.sh update   get the newest settings from GitHub and restart
set -euo pipefail
cd "$(dirname "$0")"

blue()  { printf '\033[1;34m%s\033[0m\n' "$*"; }
green() { printf '\033[1;32m%s\033[0m\n' "$*"; }
red()   { printf '\033[1;31m%s\033[0m\n' "$*"; }

ask() { # ask "question" default
  local answer
  read -r -p "$1 [$2]: " answer
  echo "${answer:-$2}"
}

need_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    red "Docker is not installed."
    echo "Install it with:  curl -fsSL https://get.docker.com | sudo sh"
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    red "Docker is installed but this user cannot use it."
    echo "Run:  sudo usermod -aG docker \$USER   then log out and back in (or run: newgrp docker)"
    exit 1
  fi
}

server_ip() {
  hostname -I 2>/dev/null | awk '{print $1}'
}

set_env() { # set_env KEY VALUE
  if grep -q "^$1=" .env; then
    sed -i "s#^$1=.*#$1=$2#" .env
  else
    echo "$1=$2" >> .env
  fi
}

if [ "${1:-}" = "update" ]; then
  need_docker
  blue "Getting the newest EARTHROPY settings…"
  git pull --ff-only
  docker compose pull
  docker compose up -d
  green "Updated. Open: http://$(server_ip):$(grep '^EARTHROPY_PORT=' .env | cut -d= -f2)"
  exit 0
fi

need_docker
blue "EARTHROPY installer"
echo

if [ ! -f .env ]; then
  cp .env.example .env
fi

github_user=$(ask "GitHub user that publishes the images" "$(grep '^EARTHROPY_GITHUB_USER=' .env | cut -d= -f2)")
set_env EARTHROPY_GITHUB_USER "$(echo "$github_user" | tr '[:upper:]' '[:lower:]')"

files=$(ask "Folder with your files" "$(grep '^EARTHROPY_FILES=' .env | cut -d= -f2)")
if [ ! -d "$files" ]; then
  red "The folder $files does not exist. Create it or choose another one."
  exit 1
fi
set_env EARTHROPY_FILES "$files"

port=$(ask "Port" "$(grep '^EARTHROPY_PORT=' .env | cut -d= -f2)")
set_env EARTHROPY_PORT "$port"

# bring data from the first EARTHROPY setup (folder with appdata/ and drive-data/)
old=$(ask "Old EARTHROPY folder to bring data from (or 'none')" "none")
if [ "$old" != "none" ]; then
  old="${old/#\~/$HOME}"
  if [ ! -d "$old" ]; then
    red "The folder $old does not exist."
    exit 1
  fi
  ( cd "$old" && docker compose down ) || true
  mkdir -p data/dashboard data/drive
  if [ -d "$old/appdata" ]; then
    cp -a "$old/appdata/." data/dashboard/
    green "  Dashboard data copied"
  fi
  if [ -d "$old/drive-data" ]; then
    find "$old/drive-data" -maxdepth 1 -type f ! -name 'config.yaml' -exec cp -a {} data/drive/ \;
    green "  A Drive data copied"
  fi
  # the dashboard needs the same key to read what it saved before
  old_key=$(grep '^SECRET_ENCRYPTION_KEY=' "$old/.env" 2>/dev/null | cut -d= -f2 || true)
  if [ -n "$old_key" ]; then
    set_env SECRET_ENCRYPTION_KEY "$old_key"
  fi
fi

if ! grep -q '^SECRET_ENCRYPTION_KEY=.\+' .env; then
  set_env SECRET_ENCRYPTION_KEY "$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
fi

# your copy of the A Drive settings (kept out of git, so updates never overwrite it)
if [ ! -f config/drive/config.yaml ]; then
  cp config/drive/config.example.yaml config/drive/config.yaml
fi

if grep -q 'CHANGE_THIS_PASSWORD' config/drive/config.yaml; then
  echo
  read -r -s -p "Choose the A Drive admin password (5+ characters): " drive_pass
  echo
  if [ "${#drive_pass}" -lt 5 ]; then
    red "Too short."
    exit 1
  fi
  escaped=$(printf '%s' "$drive_pass" | sed -e 's/[\/&#]/\\&/g' -e 's/"/\\"/g')
  sed -i "s#CHANGE_THIS_PASSWORD#${escaped}#" config/drive/config.yaml
fi

mkdir -p data/dashboard data/drive

blue "Downloading and starting the apps (the first time takes a few minutes)…"
docker compose pull
docker compose up -d

echo
green "EARTHROPY is running!"
echo "  Open:   http://$(server_ip):${port}"
echo "  A Drive login: user admin + the password you chose"
echo "  System page:   http://$(server_ip):${port}/system/"
