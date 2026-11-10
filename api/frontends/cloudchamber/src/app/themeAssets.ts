import type { AppThemeId } from './themes'

const loaded = new Set<AppThemeId>(['neutral'])

async function importPack(id: AppThemeId) {
  switch (id) {
    case 'neutral':
      await Promise.all([
        import('@astryxdesign/theme-neutral/fonts.css'),
        import('@astryxdesign/theme-neutral/theme.css'),
      ])
      break
    case 'stone':
      await Promise.all([
        import('@astryxdesign/theme-stone/fonts.css'),
        import('@astryxdesign/theme-stone/theme.css'),
      ])
      break
    case 'matcha':
      await Promise.all([
        import('@astryxdesign/theme-matcha/fonts.css'),
        import('@astryxdesign/theme-matcha/theme.css'),
      ])
      break
    default:
      break
  }
}

/** Load theme pack CSS when user selects a non-default pack (neutral ships in index.css). */
export async function ensureThemePackAssets(id: AppThemeId) {
  if (loaded.has(id)) return
  await importPack(id)
  loaded.add(id)
}
