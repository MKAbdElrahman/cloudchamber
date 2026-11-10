import { afterEach, describe, expect, it } from 'vitest'

import { APPEARANCE_STORAGE_KEY, readAppearance, writeAppearance } from './appearanceStorage'

const storage = new Map<string, string>()

afterEach(() => {
  storage.clear()
})

function installMockStorage() {
  const mock = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value)
    },
    removeItem: (key: string) => {
      storage.delete(key)
    },
  }
  Object.defineProperty(globalThis, 'localStorage', { value: mock, configurable: true })
}

describe('appearanceStorage', () => {
  it('round-trips v1 snapshot', () => {
    installMockStorage()
    writeAppearance({ v: 1, themeId: 'stone', colorMode: 'dark' })
    const read = readAppearance()
    expect(read.themeId).toBe('stone')
    expect(read.colorMode).toBe('dark')
    expect(localStorage.getItem(APPEARANCE_STORAGE_KEY)).toContain('"v":1')
  })

  it('migrates legacy keys into v1', () => {
    installMockStorage()
    localStorage.setItem('cloudchamber.astryx-theme', 'matcha')
    localStorage.setItem('cloudchamber.astryx-color-mode', 'light')
    const read = readAppearance()
    expect(read.themeId).toBe('matcha')
    expect(read.colorMode).toBe('light')
    expect(localStorage.getItem(APPEARANCE_STORAGE_KEY)).not.toBeNull()
  })
})
