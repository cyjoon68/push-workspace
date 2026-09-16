import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import {
  MemoryRouter,
  Route,
  Routes,
  createMemoryRouter,
  RouterProvider,
  useNavigate,
} from 'react-router-dom'
import { ROUTE_CONFIG } from './routes'
import { useSessionStore } from '@/shared/auth/session'
import { DocCard } from '@/shared/ui'

const seedSession = () =>
  useSessionStore.setState({
    session: {
      accessToken: 'test-token',
      refreshToken: '',
      expiresIn: 900,
      user: { id: 'u1', displayName: 'U', locale: 'ko' },
    },
  })

describe('routes', () => {
  it('maps /documents/:id to the document editor', async () => {
    seedSession()
    const router = createMemoryRouter(ROUTE_CONFIG, {
      initialEntries: ['/documents/doc-1'],
    })
    render(<RouterProvider router={router} />)
    expect(
      await screen.findByText('문서', {}, { timeout: 5000 }),
    ).toBeInTheDocument()
  })

  it('redirects to /login when signed out', async () => {
    useSessionStore.setState({ session: null })
    const router = createMemoryRouter(ROUTE_CONFIG, {
      initialEntries: ['/documents'],
    })
    render(<RouterProvider router={router} />)
    expect(await screen.findByText('Push')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/login')
  })

  it('DocCard navigates to the editor route instead of expanding inline', async () => {
    const List = () => {
      const navigate = useNavigate()
      return (
        <DocCard
          title="이력서_2026"
          meta="이력서"
          onOpen={() => navigate('/documents/doc-9')}
        />
      )
    }
    render(
      <MemoryRouter initialEntries={['/documents']}>
        <Routes>
          <Route path="/documents" element={<List />} />
          <Route
            path="/documents/:id"
            element={<div data-testid="doc-editor" />}
          />
        </Routes>
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByText('이력서_2026'))
    expect(await screen.findByTestId('doc-editor')).toBeInTheDocument()
  })
})
