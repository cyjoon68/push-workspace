import { create } from 'zustand'
import type { Conversation } from '@/shared/api'
import { createConversation, listConversations } from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type ConversationsState = {
  items: Conversation[]
  status: LoadStatus
  error: string | null
  load: () => Promise<void>
  create: (body: {
    applicationId: string | null
    title?: string
  }) => Promise<Conversation>
}

export const useConversationsStore = create<ConversationsState>()(
  (set, get) => ({
    items: [],
    status: 'idle',
    error: null,
    load: async () => {
      if (get().status === 'loading') return
      set({ status: 'loading', error: null })
      try {
        const env = await listConversations({ limit: 50 })
        set({ items: env.data, status: 'success' })
      } catch (error) {
        set({
          status: 'error',
          error: error instanceof Error ? error.message : 'error',
        })
      }
    },
    create: async (body) => {
      const created = await createConversation(body)
      set((s) => ({ items: [created, ...s.items] }))
      return created
    },
  }),
)

export const isProjectConversation = (
  conversation: Conversation,
): boolean => conversation.projectId != null
