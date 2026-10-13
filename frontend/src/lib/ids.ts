// Same format as model.NewID in Go: prefix + "_" + 12 random hex chars.
export function newId(prefix: string): string {
  const bytes = new Uint8Array(6)
  crypto.getRandomValues(bytes)
  return prefix + '_' + Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}
