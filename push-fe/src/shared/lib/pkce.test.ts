import { createHash, randomBytes } from 'node:crypto'
import { describe, expect, it } from 'vitest'
import {
  createPkceChallenge,
  createPkcePair,
  createPkceVerifier,
} from './pkce'

const referenceS256 = (verifier: string): string =>
  createHash('sha256').update(verifier).digest('base64url')

describe('pkce', () => {
  it('verifier is 64 chars from the allowed charset', () => {
    const verifier = createPkceVerifier()
    expect(verifier).toHaveLength(64)
    expect(verifier).toMatch(/^[A-Za-z0-9\-._~]+$/)
  })

  it('challenge equals base64url sha256 of verifier', async () => {
    const verifier = randomBytes(48).toString('base64url')
    expect(await createPkceChallenge(verifier)).toBe(referenceS256(verifier))
  })

  it('pair round-trips verifier to challenge', async () => {
    const { verifier, challenge } = await createPkcePair()
    expect(challenge).toBe(referenceS256(verifier))
  })
})
