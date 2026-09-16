import { create } from 'zustand'
import type { Approval, ApprovalSummary } from '@/shared/api'
import { decideApproval, getApproval } from '@/shared/api'

type ApprovalsState = {
  byId: Record<string, Approval>
  summaries: Record<string, ApprovalSummary['targetSummary']>
  ensure: (id: string) => Promise<Approval | null>
  decide: (id: string, decision: 'APPROVED' | 'DENIED') => Promise<void>
}

export const useApprovalsStore = create<ApprovalsState>()((set, get) => ({
  byId: {},
  summaries: {},
  ensure: async (id) => {
    const cached = get().byId[id]
    if (cached) return cached
    try {
      const summary = await getApproval(id)
      set((s) => ({
        byId: { ...s.byId, [id]: summary.approval },
        summaries: { ...s.summaries, [id]: summary.targetSummary },
      }))
      return summary.approval
    } catch {
      return null
    }
  },
  decide: async (id, decision) => {
    const approval = get().byId[id]
    if (!approval) return
    const updated = await decideApproval(id, {
      expectedRevision: approval.revision,
      decision,
    })
    set((s) => ({ byId: { ...s.byId, [id]: updated } }))
  },
}))
