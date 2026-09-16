import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import '@/shared/design/tokens.css'
import '@/shared/design/components.css'
import './styles.css'
import { IconSprite } from '@/shared/ui'
import { useSessionStore } from '@/shared/auth/session'
import { ROUTE_CONFIG } from './routes'

const devToken = import.meta.env.VITE_DEV_AUTH_TOKEN as string | undefined
if (devToken) {
  useSessionStore.setState({
    session: {
      accessToken: devToken,
      refreshToken: '',
      expiresIn: 900,
      user: { id: 'dev', displayName: 'Dev', locale: 'ko' },
    },
  })
}

const router = createBrowserRouter(ROUTE_CONFIG)

const handleDeepLink = (urls: string[]) => {
  for (const raw of urls) {
    if (!raw.startsWith('push://')) continue
    const url = new URL(raw.replace('push://', 'app://'))
    if (url.host === 'auth' && url.pathname === '/callback') {
      void router.navigate(`/auth/callback${url.search}`)
    }
  }
}

if ('__TAURI_INTERNALS__' in window) {
  void import('@tauri-apps/plugin-deep-link').then(({ getCurrent, onOpenUrl }) => {
    void getCurrent().then((urls) => urls && handleDeepLink(urls))
    void onOpenUrl(handleDeepLink)
  })
}

const App = () => (
  <>
    <IconSprite />
    <RouterProvider router={router} />
  </>
)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
