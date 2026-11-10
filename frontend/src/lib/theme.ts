import { WindowSetBackgroundColour } from '../../wailsjs/runtime/runtime'
import type { Theme } from './types'

const media = window.matchMedia('(prefers-color-scheme: dark)')
let current: Theme = 'system'

function resolve(): 'light' | 'dark' {
  if (current === 'system') return media.matches ? 'dark' : 'light'
  return current
}

function apply() {
  document.documentElement.dataset.theme = resolve()
  syncWindowBackground()
}

// Keeps the native window background (set at start in main.go) in line with
// the theme: it shows for an instant whenever the window is resized, for
// example when it is maximised.
function syncWindowBackground() {
  const bg = getComputedStyle(document.documentElement).getPropertyValue('--bg').trim()
  const m = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(bg)
  if (m) WindowSetBackgroundColour(parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16), 255)
}

media.addEventListener('change', () => {
  if (current === 'system') apply()
})

export function applyTheme(theme: Theme) {
  current = theme
  apply()
}
