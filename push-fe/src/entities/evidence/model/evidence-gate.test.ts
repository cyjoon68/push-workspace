import { describe, expect, it } from 'vitest'
import type { DocumentBlock } from '@/shared/api'
import {
  canFinalizeDocument,
  isBlockSupported,
  linkedEvidenceCount,
  unsupportedBlocks,
} from './evidence-gate'

const block = (
  id: string,
  claimStatus: DocumentBlock['claimStatus'],
  evidenceIds: string[] = [],
): DocumentBlock => ({
  id,
  text: `text-${id}`,
  claimStatus,
  evidenceRefs: evidenceIds.map((evidenceId) => ({
    evidenceId,
    start: 0,
    end: 1,
  })),
})

describe('evidence gate', () => {
  it('blocks finalization when a block is UNSUPPORTED', () => {
    const blocks = [
      block('a', 'SUPPORTED', ['e1']),
      block('b', 'UNSUPPORTED'),
    ]
    expect(canFinalizeDocument(blocks)).toBe(false)
  })

  it('blocks finalization when a block needs review', () => {
    expect(
      canFinalizeDocument([block('a', 'NEEDS_REVIEW', ['e1'])]),
    ).toBe(false)
  })

  it('blocks SUPPORTED claims that lost their evidence link', () => {
    expect(isBlockSupported(block('a', 'SUPPORTED'))).toBe(false)
    expect(canFinalizeDocument([block('a', 'SUPPORTED')])).toBe(false)
  })

  it('passes when every block is supported by evidence', () => {
    const blocks = [block('a', 'SUPPORTED', ['e1']), block('b', 'SUPPORTED', ['e2'])]
    expect(canFinalizeDocument(blocks)).toBe(true)
  })

  it('refuses to finalize a document with no blocks', () => {
    expect(canFinalizeDocument([])).toBe(false)
  })

  it('counts distinct linked evidence', () => {
    const blocks = [
      block('a', 'SUPPORTED', ['e1', 'e2']),
      block('b', 'SUPPORTED', ['e2', 'e3']),
    ]
    expect(linkedEvidenceCount(blocks)).toBe(3)
  })

  it('lists only unsupported blocks', () => {
    const blocks = [block('a', 'SUPPORTED', ['e1']), block('b', 'UNSUPPORTED')]
    expect(unsupportedBlocks(blocks).map((b) => b.id)).toEqual(['b'])
  })
})
