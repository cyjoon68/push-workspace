import { describe, expect, it } from 'vitest'
import {
  canTransition,
  isTerminalStage,
  nextStage,
  transitionStage,
} from './application-stage'

describe('application stage machine', () => {
  it('moves forward one stage at a time', () => {
    expect(nextStage('DISCOVERED')).toBe('PREPARING')
    expect(nextStage('READY')).toBe('APPLIED')
    expect(nextStage('OFFER')).toBe('ACCEPTED')
  })

  it('rejects skipping stages', () => {
    expect(canTransition('DISCOVERED', 'APPLIED')).toBe(false)
    expect(transitionStage('DISCOVERED', 'INTERVIEW')).toBeNull()
  })

  it('allows exit to REJECTED or WITHDRAWN from non-terminal stages', () => {
    expect(canTransition('PREPARING', 'REJECTED')).toBe(true)
    expect(canTransition('INTERVIEW', 'WITHDRAWN')).toBe(true)
  })

  it('locks terminal stages', () => {
    expect(isTerminalStage('ACCEPTED')).toBe(true)
    expect(isTerminalStage('REJECTED')).toBe(true)
    expect(canTransition('ACCEPTED', 'REJECTED')).toBe(false)
    expect(canTransition('REJECTED', 'DISCOVERED')).toBe(false)
  })

  it('rejects self-transition', () => {
    expect(canTransition('READY', 'READY')).toBe(false)
  })
})
