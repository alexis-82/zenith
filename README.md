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
