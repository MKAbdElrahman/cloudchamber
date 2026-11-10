import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LinkProvider } from '@astryxdesign/core/Link'

import { applyAppearanceToDocument, ensureThemePackAssets, readAppearance } from '@/shared/theme'
import { AstryxProviders } from '@/app/AstryxProviders'
import { RouterLink } from '@/app/RouterLink'
import { SessionProvider } from '@/shared/auth/session'
import App from './App'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 10_000,
    },
  },
})

function renderApp() {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <BrowserRouter>
        <LinkProvider component={RouterLink}>
          <AstryxProviders>
            <QueryClientProvider client={queryClient}>
              <SessionProvider>
                <App />
              </SessionProvider>
            </QueryClientProvider>
          </AstryxProviders>
        </LinkProvider>
      </BrowserRouter>
    </StrictMode>,
  )
}

/** Load appearance + theme CSS before paint (avoids theme flash and late layout shift). */
async function boot() {
  const appearance = readAppearance()
  applyAppearanceToDocument(appearance)
  await ensureThemePackAssets(appearance.themeId)
  renderApp()
}

void boot()
