import { useEffect, useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { exchangeCode } from '@/shared/api'
import { useSessionStore } from '@/shared/auth/session'
import { ErrorState, Skeleton } from '@/shared/ui'

const AuthCallbackPage = () => {
  const [params] = useSearchParams()
  const session = useSessionStore((s) => s.session)
  const setSession = useSessionStore((s) => s.setSession)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    const code = params.get('code')
    if (!code || session) return
    exchangeCode(code)
      .then(setSession)
      .catch(() => setFailed(true))
  }, [params, session, setSession])

  if (session) return <Navigate to="/" replace />
  if (failed) return <ErrorState onRetry={() => setFailed(false)} />
  return (
    <div className="canvas">
      <div className="center">
        <Skeleton />
      </div>
    </div>
  )
}

export default AuthCallbackPage
