import { useState } from 'react'

export type LibraryView = 'grid' | 'list'

const STORAGE_KEY = 'bubble.library.view.v1'

function readView(): LibraryView {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'grid' || stored === 'list') return stored
  } catch {
    // Private browsing can reject storage. The default view still works.
  }
  return 'list'
}

/** Grid or list for the project gallery. Files inside a project stay rows. */
export function useLibraryView(): [LibraryView, (next: LibraryView) => void] {
  const [view, setView] = useState<LibraryView>(readView)

  function update(next: LibraryView) {
    setView(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // The choice still applies on this page.
    }
  }

  return [view, update]
}
