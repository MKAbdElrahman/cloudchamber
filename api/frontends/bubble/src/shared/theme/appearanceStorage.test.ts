import { afterEach, describe, expect, it } from 'vitest'

import {
  APPEARANCE_STORAGE_KEY,
  applyAppearanceToDocument,
  readAppearance,
  writeAppearance,
} from './appearanceStorage'

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

  it('sets data-theme for an explicit mode and clears an inline color-scheme pin', () => {
    const attrs = new Map<string, string>()
    const root = {
      style: {
        colorScheme: 'dark',
        removeProperty(name: string) {
          if (name === 'color-scheme') this.colorScheme = ''
        },
      },
      setAttribute(name: string, value: string) {
        attrs.set(name, value)
      },
      removeAttribute(name: string) {
        attrs.delete(name)
      },
      getAttribute(name: string) {
        return attrs.get(name) ?? null
      },
    }
    Object.defineProperty(globalThis, 'document', {
      value: { documentElement: root },
      configurable: true,
    })

    applyAppearanceToDocument({ v: 1, themeId: 'neutral', colorMode: 'light' })
    expect(root.getAttribute('data-theme')).toBe('light')
    expect(root.getAttribute('data-astryx-theme')).toBe('neutral')
    expect(root.style.colorScheme).toBe('')

    applyAppearanceToDocument({ v: 1, themeId: 'stone', colorMode: 'system' })
    expect(root.getAttribute('data-theme')).toBeNull()
    expect(root.getAttribute('data-astryx-theme')).toBe('stone')
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
