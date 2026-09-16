import { create } from 'zustand'

export type SessionUser = {
  id: string
  displayName: string
  locale: string
}

export type Session = {
  accessToken: string
  refreshToken: string
  expiresIn: number
  user: SessionUser
}

type SessionState = {
  session: Session | null
  setSession: (session: Session) => void
  clearSession: () => void
}

const devToken = import.meta.env.VITE_DEV_AUTH_TOKEN as string | undefined

const devSession: Session | null = devToken
  ? {
      accessToken: devToken,
      refreshToken: '',
      expiresIn: 900,
      user: { id: 'dev', displayName: 'Dev User', locale: 'ko' },
    }
  : null

export const useSessionStore = create<SessionState>()((set) => ({
  session: devSession,
  setSession: (session) => set({ session }),
  clearSession: () => set({ session: null }),
}))

export const getAccessToken = (): string | null =>
  useSessionStore.getState().session?.accessToken ?? null
