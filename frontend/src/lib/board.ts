// Pure board mutations: every function returns a new board and never
// modifies its input. Invalid operations throw an Error with an Italian
// message meant for the user.
import { newId } from './ids'
import type { Board, Project, Task } from './types'

function cleanName(name: string, what: string): string {
  const n = name.trim()
  if (!n) throw new Error(`Il nome ${what} non può essere vuoto`)
  return n
}

function mapProject(board: Board, projectId: string, fn: (p: Project) => Project): Board {
  if (!board.projects.some((p) => p.id === projectId)) throw new Error('Progetto non trovato')
  return { ...board, projects: board.projects.map((p) => (p.id === projectId ? fn(p) : p)) }
}

// --- projects ---

export function addProject(board: Board, name: string): { board: Board; id: string } {
  const id = newId('p')
  const project: Project = { id, name: cleanName(name, 'del progetto'), tasks: [] }
  return { board: { ...board, projects: [...board.projects, project] }, id }
}

export function renameProject(board: Board, projectId: string, name: string): Board {
  const n = cleanName(name, 'del progetto')
  return mapProject(board, projectId, (p) => ({ ...p, name: n }))
}

export function deleteProject(board: Board, projectId: string): Board {
  return { ...board, projects: board.projects.filter((p) => p.id !== projectId) }
}

// --- columns (global) ---

export function addColumn(board: Board, name: string): Board {
  return { ...board, columns: [...board.columns, { id: newId('c'), name: cleanName(name, 'della colonna') }] }
}

export function renameColumn(board: Board, columnId: string, name: string): Board {
  const n = cleanName(name, 'della colonna')
  return { ...board, columns: board.columns.map((c) => (c.id === columnId ? { ...c, name: n } : c)) }
}

export function moveColumn(board: Board, columnId: string, dir: -1 | 1): Board {
  const i = board.columns.findIndex((c) => c.id === columnId)
  const j = i + dir
  if (i < 0 || j < 0 || j >= board.columns.length) return board
  const columns = [...board.columns]
  ;[columns[i], columns[j]] = [columns[j], columns[i]]
  return { ...board, columns }
}

/** Number of tasks in a column across all projects. */
export function countTasksInColumn(board: Board, columnId: string): number {
  return board.projects.reduce((n, p) => n + p.tasks.filter((t) => t.columnId === columnId).length, 0)
}

/** Deletes a column and its tasks in every project. The last column cannot be deleted. */
export function deleteColumn(board: Board, columnId: string): { board: Board; removed: number } {
  if (board.columns.length <= 1) throw new Error("Non puoi eliminare l'ultima colonna")
  const removed = countTasksInColumn(board, columnId)
  return {
    board: {
      ...board,
      columns: board.columns.filter((c) => c.id !== columnId),
      projects: board.projects.map((p) => ({ ...p, tasks: p.tasks.filter((t) => t.columnId !== columnId) })),
    },
    removed,
  }
}

// --- tasks ---

/** Tasks of a project in a column, sorted by order. */
export function tasksInColumn(project: Project, columnId: string): Task[] {
  return project.tasks.filter((t) => t.columnId === columnId).sort((a, b) => a.order - b.order)
}

/** Adds a task at the bottom of the given column. */
export function addTask(board: Board, projectId: string, columnId: string, title: string, description: string): Board {
  if (!board.columns.some((c) => c.id === columnId)) {
    throw new Error('La colonna non esiste più (forse è stata eliminata da un altro dispositivo)')
  }
  const t = cleanName(title, 'del task')
  return mapProject(board, projectId, (p) => {
    const last = tasksInColumn(p, columnId).at(-1)
    const task: Task = { id: newId('t'), title: t, description, columnId, order: last ? last.order + 1 : 0 }
    return { ...p, tasks: [...p.tasks, task] }
  })
}

export function updateTask(board: Board, projectId: string, taskId: string, title: string, description: string): Board {
  const t = cleanName(title, 'del task')
  return mapProject(board, projectId, (p) => {
    if (!p.tasks.some((x) => x.id === taskId)) {
      throw new Error('Il task non esiste più (forse è stato eliminato da un altro dispositivo)')
    }
    return { ...p, tasks: p.tasks.map((x) => (x.id === taskId ? { ...x, title: t, description } : x)) }
  })
}

export function deleteTask(board: Board, projectId: string, taskId: string): Board {
  return mapProject(board, projectId, (p) => ({ ...p, tasks: p.tasks.filter((x) => x.id !== taskId) }))
}

/**
 * Sets the content of a column after a drag & drop: every listed task is
 * moved to columnId and renumbered 0..n-1 in the given order.
 */
export function applyColumnOrder(board: Board, projectId: string, columnId: string, orderedTaskIds: string[]): Board {
  const position = new Map(orderedTaskIds.map((id, i) => [id, i]))
  return mapProject(board, projectId, (p) => ({
    ...p,
    tasks: p.tasks.map((t) => (position.has(t.id) ? { ...t, columnId, order: position.get(t.id)! } : t)),
  }))
}
