// Plain mirrors of the Go types in internal/model and internal/config.

export interface Column {
  id: string
  name: string
}

export interface Task {
  id: string
  title: string
  description: string
  columnId: string
  order: number
}

export interface Project {
  id: string
  name: string
  tasks: Task[]
}

export interface Board {
  version: number
  columns: Column[]
  projects: Project[]
}

export type Theme = 'system' | 'light' | 'dark'

export interface Settings {
  dataPath: string
  autosaveSeconds: number
  theme: Theme
  checkUpdates: boolean
}

/** Mirror of UpdateInfo in app_update.go. */
export interface UpdateInfo {
  current: string
  latest: string
  available: boolean
  canInstall: boolean
  noRelease: boolean
  notes: string
}

export interface InitState {
  settings: Settings
  board: Board | null
  loadError: string
}

export type SaveStatus = 'saved' | 'dirty' | 'saving' | 'error' | 'paused'
