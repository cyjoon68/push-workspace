import type { DocumentBlock } from '@/shared/api'

export const linkedEvidenceCount = (blocks: DocumentBlock[]): number => {
  const ids = new Set<string>()
  for (const block of blocks) {
    for (const ref of block.evidenceRefs) ids.add(ref.evidenceId)
  }
  return ids.size
}

export const isBlockSupported = (block: DocumentBlock): boolean =>
  block.claimStatus === 'SUPPORTED' && block.evidenceRefs.length > 0

export const unsupportedBlocks = (
  blocks: DocumentBlock[],
): DocumentBlock[] => blocks.filter((b) => !isBlockSupported(b))

export const canFinalizeDocument = (blocks: DocumentBlock[]): boolean =>
  blocks.length > 0 && blocks.every(isBlockSupported)
