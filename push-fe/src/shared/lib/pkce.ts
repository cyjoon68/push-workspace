const PKCE_VERIFIER_LENGTH = 64
const PKCE_CHARSET =
  'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~'

const base64UrlEncode = (bytes: Uint8Array): string =>
  btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')

export const createPkceVerifier = (): string => {
  const random = crypto.getRandomValues(new Uint8Array(PKCE_VERIFIER_LENGTH))
  return Array.from(random, (b) => PKCE_CHARSET[b % PKCE_CHARSET.length]).join(
    '',
  )
}

export const createPkceChallenge = async (
  verifier: string,
): Promise<string> => {
  const digest = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(verifier),
  )
  return base64UrlEncode(new Uint8Array(digest))
}

export type PkcePair = {
  verifier: string
  challenge: string
}

export const createPkcePair = async (): Promise<PkcePair> => {
  const verifier = createPkceVerifier()
  return { verifier, challenge: await createPkceChallenge(verifier) }
}
