import { useEffect } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { useSessionStore } from '@/shared/auth/session'
import { ToastHost } from '@/shared/ui'
import { useInferenceSettings } from '@/features/chat'
import Sidebar from './sidebar'

const AppShell = () => {
  const session = useSessionStore((s) => s.session)
  const loadModels = useInferenceSettings((s) => s.loadModels)

  useEffect(() => {
    void loadModels()
  }, [loadModels])

  if (!session) return <Navigate to="/login" replace />

  return (
    <div className="app-shell">
      <Sidebar />
      <div className="canvas">
        <Outlet />
      </div>
      <ToastHost />
    </div>
  )
}

export default AppShell
