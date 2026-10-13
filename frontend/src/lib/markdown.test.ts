// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { isSafeUrl, renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  it('renders basic markdown', () => {
    const html = renderMarkdown('**grassetto**\n\n- uno\n- due\n\n`codice` [link](https://example.com)')
    expect(html).toContain('<strong>grassetto</strong>')
    expect(html).toContain('<li>uno</li>')
    expect(html).toContain('<code>codice</code>')
    expect(html).toContain('href="https://example.com"')
  })

  it('strips scripts and event handlers', () => {
    const html = renderMarkdown('<img src=x onerror=alert(1)>\n\n<script>alert(2)</script>\n\n<a href="javascript:alert(3)">x</a>')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('<script')
    expect(html).not.toContain('javascript:')
  })
})

describe('isSafeUrl', () => {
  it.each([
    ['https://example.com', true],
    ['http://example.com/a', true],
    ['mailto:me@example.com', true],
    ['file:///C:/Windows/System32/calc.exe', false],
    ['javascript:alert(1)', false],
    ['ms-settings:', false],
    ['steam://run/1', false],
    ['relative/path', false],
  ])('%s -> %s', (u, want) => {
    expect(isSafeUrl(u)).toBe(want)
  })
})
