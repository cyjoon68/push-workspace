import type { ApplicationStage } from '@/shared/api'
import type { StatusChipTone } from '@/shared/ui'

const STAGE_TONES: Record<ApplicationStage, StatusChipTone> = {
  DISCOVERED: 'pending',
  PREPARING: 'running',
  READY: 'ready',
  APPLIED: 'verified',
  SCREENING: 'verified',
  INTERVIEW: 'verified',
  OFFER: 'verified',
  ACCEPTED: 'verified',
  REJECTED: 'error',
  WITHDRAWN: 'error',
}

export const stageChipTone = (stage: ApplicationStage): StatusChipTone =>
  STAGE_TONES[stage]
