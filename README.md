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

## Note sul cloud

- Zenith legge e scrive solo il file locale; la sincronizzazione la fa il client Nextcloud.
- Se due computer modificano il file mentre sono offline, Nextcloud può creare una copia `… (conflicted copy …).json`. Zenith la ignora: se serve, confronta e unisci a mano.
- Su Linux con WebKitGTK datato il tema "Sistema" potrebbe non seguire il tema scuro: in quel caso scegli "Scuro".

### Linux: pacchetto .deb (Debian, Ubuntu, Mint)

Dalla pagina delle release scarica `zenith_<versione>_amd64.deb` e installalo (apt installa anche le dipendenze GTK/WebKitGTK):

```sh
sudo apt install ./zenith_1.2.0_amd64.deb
```

Installa `/usr/bin/zenith`, il launcher `/usr/bin/zenith-launch`, la voce di menu e l'icona. Per disinstallare: `sudo apt remove zenith` (il file dati JSON e `~/.config/Zenith` restano intatti).

Per creare il pacchetto a mano (su Linux, dalla root del progetto) serve [nfpm](https://nfpm.goreleaser.com): `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0`.

```sh
wails build -tags webkit2_41 -ldflags "-X main.version=v1.2.0"
VERSION=1.2.0 ARCH=amd64 nfpm pkg --config build/linux/nfpm.yaml --packager deb --target build/bin/
```

La configurazione è in `build/linux/nfpm.yaml`; nelle release il pacchetto lo crea il workflow `release.yml`.

### Linux: pacchetto .rpm (Fedora 40+)

Dalla pagina delle release scarica `zenith-<versione>-1.x86_64.rpm` e installalo (dnf installa anche le dipendenze GTK/WebKitGTK):

```sh
sudo dnf install ./zenith-1.2.0-1.x86_64.rpm
```

Installa gli stessi file del `.deb` (`/usr/bin/zenith`, `/usr/bin/zenith-launch`, la voce di menu e l'icona). Per disinstallare: `sudo dnf remove zenith` (il file dati JSON e `~/.config/Zenith` restano intatti). Serve Fedora 40 o successiva: il programma è compilato su Ubuntu 24.04 e richiede la stessa glibc o più recente.

Per creare il pacchetto a mano serve nfpm, come per il `.deb`:

```sh
wails build -tags webkit2_41 -ldflags "-X main.version=v1.2.0"
VERSION=1.2.0 ARCH=amd64 nfpm pkg --config build/linux/nfpm.yaml --packager rpm --target build/bin/
```

### Linux: installazione con script (altre distribuzioni)

Dalla root del progetto, dopo la build:

```sh
wails build -tags webkit2_41          # Ubuntu 24.04+, Debian 13, Fedora 40+ (webkit2gtk 4.1)
sudo sh build/linux/install.sh
```

L'installer copia:

- `build/bin/zenith` in `/usr/local/bin/zenith`;
- il launcher `build/linux/zenith-launch.sh` come `/usr/local/bin/zenith-launch`;
- la voce di menu `zenith.desktop` in `/usr/share/applications/`;
- l'icona in `/usr/share/pixmaps/zenith.png`;
- un collegamento `zenith.desktop` sul Desktop dell'utente che ha lanciato `sudo`, segnato come attendibile (GNOME non avvia i collegamenti non attendibili).

Per aggiornare basta rifare build e install. Per disinstallare (il file dati JSON e `~/.config/Zenith` restano intatti):

```sh
sudo sh build/linux/install.sh --uninstall
```

Il launcher imposta `WEBKIT_DISABLE_DMABUF_RENDERER=1`, che evita la finestra bianca di WebKitGTK con alcuni driver (NVIDIA, alcune sessioni Wayland). Per non usarlo: `WEBKIT_DISABLE_DMABUF_RENDERER=0 zenith-launch`.

Se il collegamento sul desktop non si avvia, sceglierlo con il tasto destro e selezionare **Consenti avvio** (GNOME) oppure renderlo eseguibile dal file manager.

## Aggiornamenti

Zenith confronta la propria versione con l'ultima release di <https://github.com/alexis-82/zenith/releases>: all'avvio (disattivabile in **Impostazioni → Aggiornamenti**) e con il pulsante **Controlla aggiornamenti**. Se c'è una versione più recente mostra le note di rilascio e, con **Scarica e installa**:

- **Windows**: scarica il file `*-installer.exe` della release, lo verifica, salva i dati, avvia il setup e si chiude;
- **Linux**, installato con il `.deb`: scarica `zenith_<versione>_<arch>.deb`, lo verifica e lo installa con `pkexec apt-get install` (richiesta password), poi riavvia Zenith;
- **Linux**, installato con il `.rpm`: scarica `zenith-<versione>-1.<arch>.rpm`, lo verifica e lo installa con `pkexec dnf install` (richiesta password), poi riavvia Zenith;
- **Linux**, installato con `install.sh`: scarica `zenith-linux-<arch>.tar.gz`, lo verifica e lo installa con `pkexec zenith-update.sh`, poi riavvia Zenith.

Il download è accettato solo da `github.com/alexis-82/zenith/releases/download/…` e viene verificato per dimensione e, se GitHub lo fornisce, SHA-256. Offline o senza release l'app funziona normalmente.

Su Linux l'aggiornamento si può fare anche a mano: `sudo apt install ./zenith_<versione>_amd64.deb` (pacchetto .deb), `sudo dnf install ./zenith-<versione>-1.x86_64.rpm` (pacchetto .rpm), oppure `sudo zenith-update.sh` (scarica e installa l'ultima release) oppure `sudo zenith-update.sh zenith-linux-amd64.tar.gz`.