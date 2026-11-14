import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LinkProvider } from '@astryxdesign/core/Link'
import '@astryxdesign/theme-neutral/fonts.css'
import '@astryxdesign/theme-stone/fonts.css'
import '@astryxdesign/theme-matcha/fonts.css'
import '@astryxdesign/core/reset.css'
import '@astryxdesign/core/astryx.css'
import '@astryxdesign/theme-neutral/theme.css'
import '@astryxdesign/theme-stone/theme.css'
import '@astryxdesign/theme-matcha/theme.css'

import { applyAppearanceToDocument, readAppearance } from '@/shared/theme'
import { AstryxProviders } from '@/app/AstryxProviders'
import { RouterLink } from '@/app/RouterLink'
import { SessionProvider } from '@/shared/auth'
import App from './App'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 10_000,
    },
  },
})

applyAppearanceToDocument(readAppearance())

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
