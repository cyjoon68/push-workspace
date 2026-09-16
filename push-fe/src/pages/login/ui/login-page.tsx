import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { startOAuth } from '@/shared/api'
import type { AuthProvider } from '@/shared/api'
import { useSessionStore } from '@/shared/auth/session'
import { Button, Card, ErrorState } from '@/shared/ui'

const LoginPage = () => {
  const session = useSessionStore((s) => s.session)
  const [pending, setPending] = useState<AuthProvider | null>(null)
  const [failed, setFailed] = useState(false)

  if (session) return <Navigate to="/" replace />

  const start = async (provider: AuthProvider) => {
    setPending(provider)
    setFailed(false)
    try {
      const { authorizationUrl } = await startOAuth(provider)
      window.location.assign(authorizationUrl)
    } catch {
      setFailed(true)
    } finally {
      setPending(null)
    }
  }

  return (
    <div className="canvas">
      <div className="center">
        <Card title="Push">
          <p className="t-body-sm">
            채팅으로 지원·문서·프로젝트를 관리하는 데스크톱 앱
          </p>
        </Card>
        {failed ? (
          <ErrorState onRetry={() => setFailed(false)} />
        ) : (
          <>
            <Button
              variant="primary"
              size="lg"
              loading={pending === 'google'}
              onClick={() => void start('google')}
            >
              Google로 계속
            </Button>
            <Button
              variant="secondary"
              size="lg"
              loading={pending === 'github'}
              onClick={() => void start('github')}
            >
              GitHub로 계속
            </Button>
          </>
        )}
      </div>
    </div>
  )
}

export default LoginPage
