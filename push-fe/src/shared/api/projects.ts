import { api } from './client'
import { request } from './envelope'
import type { DataEnvelope, ListEnvelope, ListParams } from './envelope'
import { newIdempotencyKey } from '../lib/id'

export type CliProvider = 'CODEX' | 'CLAUDE_CODE' | 'GROK_BUILD'

export type CliRunState =
  | 'DRAFT'
  | 'APPROVAL_REQUIRED'
  | 'RUNNING'
  | 'VERIFYING'
  | 'VERIFIED'
  | 'FAILED'

export type CliRun = {
  id: string
  revision: number
  projectId: string
  applicationId: string
  provider: CliProvider
  workingDirectory: string
  executable: string
  arguments: string[]
  prompt: string
  payloadHash: string
  state: CliRunState
  approvalId: string | null
  startedAt: string | null
  finishedAt: string | null
  failureReason: string | null
  createdAt: string
  updatedAt: string
}

export type BlueprintTask = {
  id: string
  title: string
  description: string
  acceptance: string[]
}

export type BlueprintMetric = {
  name: string
  unit: string
  measurement: string
  target: number | null
}

export type ProjectBlueprint = {
  id: string
  revision: number
  applicationId: string
  gapAnalysisId: string
  title: string
  skills: string[]
  problem: string
  solution: string
  tasks: BlueprintTask[]
  completionCriteria: string[]
  metrics: BlueprintMetric[]
  estimatedEffort: { minHours: number; maxHours: number }
  state: 'DRAFT' | 'SELECTED' | 'IN_PROGRESS' | 'VERIFIED' | 'ARCHIVED'
  createdAt: string
  updatedAt: string
}

export type ProjectEvidence = {
  id: string
  revision: number
  projectId: string
  runId: string
  commitUrl: string
  commitSha: string
  testResults: string
  metrics: { name: string; value: number; unit: string }[]
  summary: string
  status: 'PENDING' | 'VERIFIED' | 'REJECTED'
  verificationMethod: string | null
  verifiedAt: string | null
  careerEvidenceId: string | null
}

export const listProjects = async (
  params: ListParams & { applicationId?: string } = {},
): Promise<ListEnvelope<ProjectBlueprint>> =>
  request(() =>
    api.get('projects', {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
        ...(params.applicationId ? { applicationId: params.applicationId } : {}),
      },
    }),
  )

export const listProjectRuns = async (
  projectId: string,
  params: ListParams = {},
): Promise<ListEnvelope<CliRun>> =>
  request(() =>
    api.get(`projects/${projectId}/runs`, {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
      },
    }),
  )

export const createProjectRun = async (
  projectId: string,
  body: {
    provider: CliProvider
    workingDirectory: string
    prompt: string
  },
): Promise<CliRun> => {
  const env = await request<DataEnvelope<CliRun>>(() =>
    api.post(`projects/${projectId}/runs`, {
      json: body,
      headers: { 'Idempotency-Key': newIdempotencyKey() },
    }),
  )
  return env.data
}

export const listProjectEvidence = async (
  projectId: string,
  params: ListParams = {},
): Promise<ListEnvelope<ProjectEvidence>> =>
  request(() =>
    api.get(`projects/${projectId}/evidence`, {
      searchParams: {
        ...(params.limit ? { limit: params.limit } : {}),
        ...(params.cursor ? { cursor: params.cursor } : {}),
      },
    }),
  )
