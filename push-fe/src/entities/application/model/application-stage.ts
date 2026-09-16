import type { ApplicationStage } from '@/shared/api'

export const APPLICATION_PIPELINE: readonly ApplicationStage[] = [
  'DISCOVERED',
  'PREPARING',
  'READY',
  'APPLIED',
  'SCREENING',
  'INTERVIEW',
  'OFFER',
  'ACCEPTED',
] as const

export const EXIT_STAGES: readonly ApplicationStage[] = [
  'REJECTED',
  'WITHDRAWN',
] as const

export const TERMINAL_STAGES: readonly ApplicationStage[] = [
  'ACCEPTED',
  'REJECTED',
  'WITHDRAWN',
] as const

export const isTerminalStage = (stage: ApplicationStage): boolean =>
  TERMINAL_STAGES.includes(stage)

export const nextStage = (from: ApplicationStage): ApplicationStage | null => {
  const index = APPLICATION_PIPELINE.indexOf(from)
  if (index < 0 || index === APPLICATION_PIPELINE.length - 1) return null
  return APPLICATION_PIPELINE[index + 1]
}

export const canTransition = (
  from: ApplicationStage,
  to: ApplicationStage,
): boolean => {
  if (from === to) return false
  if (isTerminalStage(from)) return false
  if (to === nextStage(from)) return true
  return EXIT_STAGES.includes(to)
}

export const transitionStage = (
  from: ApplicationStage,
  to: ApplicationStage,
): ApplicationStage | null => (canTransition(from, to) ? to : null)
