![logo](https://i.ibb.co/WpPfMf2p/zenith-logo-dark.png)

# Zenith

Kanban board personale e minimale (Go + Wails + Svelte). Tutti i progetti e i task stanno in **un unico file JSON**, pensato per essere tenuto in una cartella sincronizzata (es. Nextcloud) e aperto da più computer.

## Funzionalità

- Progetti: crea, rinomina, elimina.
- Colonne globali (uguali per tutti i progetti): aggiungi, rinomina, sposta ←/→, elimina (elimina anche i task contenuti, in tutti i progetti). Default: Implementare, In corso, Terminato.
- Task con titolo e descrizione Markdown (Modifica/Anteprima), drag & drop tra colonne e su/giù.
- Salvataggio automatico N secondi dopo l'ultima modifica (configurabile) e alla chiusura.
- Avviso quando il file viene modificato da un altro dispositivo; in caso di conflitto con modifiche non salvate si sceglie quale versione tenere.
- Tema Sistema / Chiaro / Scuro.

## Primo avvio

1. Apri **Impostazioni**.
2. **Crea nuovo file…** nella cartella di Nextcloud (primo computer), oppure **Sfoglia…** per scegliere il file già sincronizzato (altri computer). In alternativa scrivi il percorso e premi **Applica**.

Su un computer nuovo, se Nextcloud non ha ancora scaricato il file, Zenith segnala che il file non esiste e **non** crea un file vuoto: attendi la sincronizzazione e riprova.

## Dove sono i dati

- File dati: quello scelto nelle Impostazioni (condiviso tra i computer).
- Impostazioni locali (percorso del file, intervallo di salvataggio, tema), per computer:
  - Windows: `%AppData%\Zenith\settings.json`
  - macOS: `~/Library/Application Support/Zenith/settings.json`
  - Linux: `~/.config/Zenith/settings.json`

## Note su Nextcloud

- Zenith legge e scrive solo il file locale; la sincronizzazione la fa il client Nextcloud.
- Se due computer modificano il file mentre sono offline, Nextcloud può creare una copia `… (conflicted copy …).json`. Zenith la ignora: se serve, confronta e unisci a mano.
- Su Linux con WebKitGTK datato il tema "Sistema" potrebbe non seguire il tema scuro: in quel caso scegli "Scuro".

## Sviluppo

Requisiti: Go ≥ 1.25, Node ≥ 20, [Wails CLI v2](https://wails.io/docs/gettingstarted/installation).

Dopo un clone, prima di usare `go build` / `go test`, compila una volta il frontend: `main.go` incorpora `frontend/dist` (non versionata) e Go non compila se la cartella non esiste. `wails build` e `wails dev` lo fanno da soli.

```sh
npm --prefix frontend install && npm --prefix frontend run build
```

```sh
wails dev                    # avvio in sviluppo
go test ./...                # test backend
npm --prefix frontend test   # test frontend
wails build                  # build per il sistema corrente -> build/bin/
```

La build va eseguita su ciascun sistema operativo:

- **Windows**: richiede WebView2 (preinstallato su Windows 10/11 aggiornati).
- **macOS**: `wails build` (Xcode command line tools).
- **Linux**: servono `gcc`, `libgtk-3-dev` e `libwebkit2gtk-4.0-dev` (o `libwebkit2gtk-4.1-dev` con `wails build -tags webkit2_41`).

### Linux: installazione nel menu e collegamento sul desktop

Dalla root del progetto, dopo la build:

```sh
wails build -tags webkit2_41          # Ubuntu 24.04+, Debian 13, Fedora 40+ (webkit2gtk 4.1)
sudo sh build/linux/install.sh
```

L'installer copia:

- `build/bin/zenith` in `/usr/local/bin/zenith`;
- il launcher `build/linux/zenith-launch.sh` in `/usr/local/bin/`;
- la voce di menu `zenith.desktop` in `/usr/share/applications/`;
- l'icona in `/usr/share/pixmaps/zenith.png`;
- un collegamento `zenith.desktop` sul Desktop dell'utente che ha lanciato `sudo`, segnato come attendibile (GNOME non avvia i collegamenti non attendibili).

Per aggiornare basta rifare build e install. Per disinstallare (il file dati JSON e `~/.config/Zenith` restano intatti):

```sh
sudo sh build/linux/install.sh --uninstall
```

Il launcher imposta `WEBKIT_DISABLE_DMABUF_RENDERER=1`, che evita la finestra bianca di WebKitGTK con alcuni driver (NVIDIA, alcune sessioni Wayland). Per non usarlo: `WEBKIT_DISABLE_DMABUF_RENDERER=0 zenith-launch.sh`.

Se il collegamento sul desktop non si avvia, sceglierlo con il tasto destro e selezionare **Consenti avvio** (GNOME) oppure renderlo eseguibile dal file manager.

## Aggiornamenti

Zenith confronta la propria versione con l'ultima release di <https://github.com/alexis-82/zenith/releases>: all'avvio (disattivabile in **Impostazioni → Aggiornamenti**) e con il pulsante **Controlla aggiornamenti**. Se c'� una versione pi� recente mostra le note di rilascio e, con **Scarica e installa**:

- **Windows**: scarica il file `*-setup.exe` della release, lo verifica, salva i dati, avvia il setup e si chiude;
- **Linux**: scarica `zenith-linux-<arch>.tar.gz`, lo verifica e lo installa con `pkexec zenith-update.sh` (richiesta password), poi riavvia Zenith.

Il download � accettato solo da `github.com/alexis-82/zenith/releases/download/…` e viene verificato per dimensione e, se GitHub lo fornisce, SHA-256. Offline o senza release l'app funziona normalmente.

Su Linux l'aggiornamento si pu� fare anche a mano: `sudo zenith-update.sh` (scarica e installa l'ultima release) oppure `sudo zenith-update.sh zenith-linux-amd64.tar.gz`.

### Pubblicare una release

La versione � il **tag** della release, nel formato `vX.Y.Z`, e va impressa in compilazione: le build senza versione risultano `dev` e non propongono aggiornamenti.

1. Scegli la versione, es. `v1.2.0`, e crea il tag:
   ```sh
   git tag v1.2.0
   git push origin v1.2.0
   ```
2. **Windows** (installer):
   ```powershell
   wails build -ldflags "-X main.version=v1.2.0"
   makensis /DVERSION=1.2.0 build\windows\installer\zenith.nsi
   ```
   Risultato: `build\bin\Zenith-1.2.0-setup.exe` (il nome deve finire con `-setup.exe`).
3. **Linux** (su una macchina Linux, per ogni architettura):
   ```sh
   wails build -tags webkit2_41 -ldflags "-X main.version=v1.2.0"
   sh build/linux/package.sh
   ```
   Risultato: `build/bin/zenith-linux-amd64.tar.gz` (o `-arm64`).
4. Su GitHub → **Releases → Draft a new release**: scegli il tag `v1.2.0`, scrivi le note (Markdown, vengono mostrate nell'app) e allega i due file. Pubblica.
   - Le release segnate come **pre-release** o **draft** non vengono proposte.
   - Con la GitHub CLI: `gh release create v1.2.0 build/bin/Zenith-1.2.0-setup.exe build/bin/zenith-linux-amd64.tar.gz --notes-file NOTE.md`
