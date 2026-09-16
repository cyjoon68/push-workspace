import { create } from 'zustand'
import type { CliRun, ProjectEvidence } from '@/shared/api'
import { listProjectEvidence, listProjectRuns } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type ProjectPanelsState = {
  projectId: string | null
  runs: CliRun[]
  evidence: ProjectEvidence[]
  status: LoadStatus
  load: (projectId: string) => Promise<void>
  reset: () => void
}

export const useProjectPanelsStore = create<ProjectPanelsState>()((set) => ({
  projectId: null,
  runs: [],
  evidence: [],
  status: 'idle',
  load: async (projectId) => {
    set({ projectId, status: 'loading', runs: [], evidence: [] })
    try {
      const [runs, evidence] = await Promise.all([
        listProjectRuns(projectId, { limit: 20 }),
        listProjectEvidence(projectId, { limit: 20 }),
      ])
      set({ runs: runs.data, evidence: evidence.data, status: 'success' })
    } catch {
      set({ status: 'error' })
    }
  },
  reset: () =>
    set({ projectId: null, runs: [], evidence: [], status: 'idle' }),
}))
