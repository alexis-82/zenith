import { describe, expect, it } from 'vitest'
import {
  addColumn,
  addProject,
  addTask,
  applyColumnOrder,
  countTasksInColumn,
  deleteColumn,
  deleteProject,
  deleteTask,
  moveColumn,
  renameColumn,
  renameProject,
  tasksInColumn,
  updateTask,
} from './board'
import { newId } from './ids'
import type { Board } from './types'

function emptyBoard(): Board {
  return {
    version: 1,
    columns: [
      { id: 'c_impl', name: 'Implementare' },
      { id: 'c_doing', name: 'In corso' },
      { id: 'c_done', name: 'Terminato' },
    ],
    projects: [],
  }
}

function withProjects(): { board: Board; p1: string; p2: string } {
  let r = addProject(emptyBoard(), 'Uno')
  const p1 = r.id
  r = addProject(r.board, 'Due')
  const p2 = r.id
  let board = addTask(r.board, p1, 'A', '')
  board = addTask(board, p1, 'B', '')
  board = addTask(board, p2, 'C', 'desc')
  return { board, p1, p2 }
}

const project = (b: Board, id: string) => b.projects.find((p) => p.id === id)!

describe('ids', () => {
  it('uses the Go format', () => {
    expect(newId('t')).toMatch(/^t_[0-9a-f]{12}$/)
    expect(newId('t')).not.toBe(newId('t'))
  })
})

describe('projects', () => {
  it('adds, renames and deletes without mutating input', () => {
    const b0 = emptyBoard()
    const { board: b1, id } = addProject(b0, '  Test  ')
    expect(b0.projects).toHaveLength(0)
    expect(project(b1, id).name).toBe('Test')

    const b2 = renameProject(b1, id, 'Test2')
    expect(project(b1, id).name).toBe('Test')
    expect(project(b2, id).name).toBe('Test2')

    expect(deleteProject(b2, id).projects).toHaveLength(0)
  })

  it('rejects empty or blank names', () => {
    expect(() => addProject(emptyBoard(), '   ')).toThrow()
    const { board, id } = addProject(emptyBoard(), 'X')
    expect(() => renameProject(board, id, '')).toThrow()
  })
})

describe('columns', () => {
  it('adds, renames and moves', () => {
    let b = addColumn(emptyBoard(), 'Revisione')
    const rev = b.columns[3].id
    b = renameColumn(b, rev, 'Review')
    b = moveColumn(b, rev, -1)
    expect(b.columns.map((c) => c.name)).toEqual(['Implementare', 'In corso', 'Review', 'Terminato'])
    expect(moveColumn(b, 'c_impl', -1)).toBe(b) // already first
    expect(() => addColumn(b, ' ')).toThrow()
    expect(() => renameColumn(b, rev, '')).toThrow()
  })

  it('deletes a column and its tasks in every project', () => {
    const { board, p1, p2 } = withProjects()
    expect(countTasksInColumn(board, 'c_impl')).toBe(3)
    const { board: b, removed } = deleteColumn(board, 'c_impl')
    expect(removed).toBe(3)
    expect(b.columns.map((c) => c.id)).toEqual(['c_doing', 'c_done'])
    expect(project(b, p1).tasks).toHaveLength(0)
    expect(project(b, p2).tasks).toHaveLength(0)
    expect(project(board, p1).tasks).toHaveLength(2) // input untouched
  })

  it('refuses to delete the last column', () => {
    const b: Board = { version: 1, columns: [{ id: 'c_only', name: 'Unica' }], projects: [] }
    expect(() => deleteColumn(b, 'c_only')).toThrow()
  })
})

describe('tasks', () => {
  it('adds at the bottom of the first column', () => {
    const { board, p1 } = withProjects()
    const tasks = tasksInColumn(project(board, p1), 'c_impl')
    expect(tasks.map((t) => [t.title, t.order])).toEqual([
      ['A', 0],
      ['B', 1],
    ])
  })

  it('updates and deletes', () => {
    const { board, p2 } = withProjects()
    const id = project(board, p2).tasks[0].id
    const b = updateTask(board, p2, id, 'Nuovo', '**md**')
    expect(project(b, p2).tasks[0]).toMatchObject({ title: 'Nuovo', description: '**md**' })
    expect(() => updateTask(board, p2, id, ' ', '')).toThrow()
    expect(() => updateTask(board, p2, 't_missing', 'X', '')).toThrow(/non esiste più/)
    expect(project(deleteTask(b, p2, id), p2).tasks).toHaveLength(0)
  })

  it('applyColumnOrder moves across columns and renumbers', () => {
    const { board, p1 } = withProjects()
    const [a, bTask] = tasksInColumn(project(board, p1), 'c_impl')
    // drag A into "In corso": both zones finalize
    let b = applyColumnOrder(board, p1, 'c_impl', [bTask.id])
    b = applyColumnOrder(b, p1, 'c_doing', [a.id])
    const p = project(b, p1)
    expect(tasksInColumn(p, 'c_impl').map((t) => [t.title, t.order])).toEqual([['B', 0]])
    expect(tasksInColumn(p, 'c_doing').map((t) => [t.title, t.order])).toEqual([['A', 0]])

    // reorder inside a column
    b = applyColumnOrder(b, p1, 'c_doing', [])
    b = applyColumnOrder(b, p1, 'c_impl', [a.id, bTask.id])
    expect(tasksInColumn(project(b, p1), 'c_impl').map((t) => t.title)).toEqual(['A', 'B'])
  })

  it('applyColumnOrder gives the same result in either finalize order', () => {
    const { board, p1 } = withProjects()
    const [a, bTask] = tasksInColumn(project(board, p1), 'c_impl')
    const x = applyColumnOrder(applyColumnOrder(board, p1, 'c_impl', [bTask.id]), p1, 'c_doing', [a.id])
    const y = applyColumnOrder(applyColumnOrder(board, p1, 'c_doing', [a.id]), p1, 'c_impl', [bTask.id])
    expect(x).toEqual(y)
  })
})
