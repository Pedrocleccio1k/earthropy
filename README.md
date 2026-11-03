# EARTHROPY

Self-hosting na mão de todo mundo — até de quem nunca instalou Linux. Um único programa que vira o seu servidor pessoal: arquivos (**A Drive**), vídeos (**Whatch**) e um painel com widgets, para você e a sua galera.

- **Um executável.** Sem Docker, sem instalar nada. Baixa, abre, usa.
- **Linux, Windows e macOS.**
- **Atualiza com um botão** (Ajustes › Atualizações), como no Linux: você escolhe quando.
- **Seus dados ficam seus**: tudo em uma pasta que você conhece.

## Como usar

1. Baixe o arquivo do seu sistema na [última release](../../releases/latest):
   `earthropy-linux-amd64`, `earthropy-windows-amd64.exe`, `earthropy-darwin-arm64` (Mac com chip Apple) ou `earthropy-darwin-amd64` (Mac Intel).
2. **Linux/macOS:** dê permissão de execução uma vez e rode:
   ```sh
   chmod +x earthropy-linux-amd64
   ./earthropy-linux-amd64
   ```
   **Windows:** dê dois cliques no `.exe`.
3. O navegador abre em `http://localhost:7777`. Na primeira vez você cria a conta do administrador.

Para a galera entrar de fora de casa, use o [Tailscale](https://tailscale.com) (Funnel) apontando para a porta 7777.

### Opções
```
-porta 7777          porta do servidor
-dados <pasta>       onde guardar banco, arquivos e lixeira
                     (padrão: ~/.local/share/earthropy no Linux; AppData no Windows; ~/Library/Application Support no macOS)
-sem-navegador       não abrir o navegador ao iniciar
```
Também vale a variável `EARTHROPY_DATA` para a pasta de dados.

## Compilar do código
```sh
go build -ldflags "-s -w" .
```

## Publicar uma versão
Crie uma tag `vX.Y.Z` e envie; o GitHub Actions compila para todos os sistemas, cria a Release e o `latest.json` que o botão **Atualizar** usa.
```sh
git tag -a v0.1.0 -m "Primeira versão: A Drive, Whatch, painel e atualização automática"
git push origin v0.1.0
```

## Licença
AGPL-3.0 — veja [LICENSE](LICENSE).
