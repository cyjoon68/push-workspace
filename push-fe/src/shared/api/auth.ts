import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope } from './envelope'
import { createPkcePair } from '@/shared/lib/pkce'
import type { Session } from '@/shared/auth/session'

export type AuthProvider = 'google' | 'github'

export const PKCE_VERIFIER_STORAGE_KEY = 'push:pkce-verifier'
export const OAUTH_REDIRECT_URI = 'push://auth/callback'

export type OAuthStart = {
  authorizationUrl: string
  state: string
  expiresAt: string
}

export type AuthUser = {
  id: string
  displayName: string
  locale: string
  createdAt: string
}

export const startOAuth = async (
  provider: AuthProvider,
): Promise<OAuthStart> => {
  const { verifier, challenge } = await createPkcePair()
  const env = await request<DataEnvelope<OAuthStart>>(() =>
    api.get(`auth/${provider}/start`, {
      searchParams: {
        redirectUri: OAUTH_REDIRECT_URI,
        codeChallenge: challenge,
        codeChallengeMethod: 'S256',
      },
    }),
  )
  sessionStorage.setItem(PKCE_VERIFIER_STORAGE_KEY, verifier)
  return env.data
}

export const exchangeCode = async (code: string): Promise<Session> => {
  const codeVerifier = sessionStorage.getItem(PKCE_VERIFIER_STORAGE_KEY) ?? ''
  const env = await request<DataEnvelope<Session>>(() =>
    api.post('auth/exchange', { json: { code, codeVerifier } }),
  )
  sessionStorage.removeItem(PKCE_VERIFIER_STORAGE_KEY)
  return env.data
}

export const fetchMe = async (): Promise<AuthUser> => {
  const env = await request<DataEnvelope<AuthUser>>(() => api.get('auth/me'))
  return env.data
}

export const logout = (refreshToken: string): Promise<void> =>
  request(() => api.post('auth/logout', { json: { refreshToken } }))
