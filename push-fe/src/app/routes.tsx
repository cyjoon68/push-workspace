import type { RouteObject } from 'react-router-dom'
import { LoginPage } from '@/pages/login'
import { AuthCallbackPage } from '@/pages/auth-callback'
import { HomePage } from '@/pages/home'
import { ChatPage } from '@/pages/chat'
import { DocumentsPage } from '@/pages/documents'
import { DocEditorPage } from '@/pages/doc-editor'
import { ApplicationsPage } from '@/pages/applications'
import { JobDetailPage } from '@/pages/job-detail'
import { VaultPage } from '@/pages/vault'
import { InterviewPage } from '@/pages/interview'
import { CalendarPage } from '@/pages/calendar'
import { SettingsPage } from '@/pages/settings'
import AppShell from './ui/app-shell'

export const ROUTE_CONFIG: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  { path: '/auth/callback', element: <AuthCallbackPage /> },
  {
    element: <AppShell />,
    children: [
      { path: '/', element: <HomePage /> },
      { path: '/chat/:id', element: <ChatPage /> },
      { path: '/documents', element: <DocumentsPage /> },
      { path: '/documents/:id', element: <DocEditorPage /> },
      { path: '/applications', element: <ApplicationsPage /> },
      { path: '/jobs/:id', element: <JobDetailPage /> },
      { path: '/vault', element: <VaultPage /> },
      { path: '/interview', element: <InterviewPage /> },
      { path: '/calendar', element: <CalendarPage /> },
      { path: '/settings', element: <SettingsPage /> },
    ],
  },
]
