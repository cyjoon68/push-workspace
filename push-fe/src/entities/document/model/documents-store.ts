import { create } from 'zustand'
import type { DocumentVersion, PushDocument } from '@/shared/api'
import {
  createDocument,
  getDocument,
  getDocumentVersion,
  listDocuments,
} from '@/shared/api'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type DocumentsState = {
  items: PushDocument[]
  status: LoadStatus
  error: string | null
  current: PushDocument | null
  currentVersion: DocumentVersion | null
  currentStatus: LoadStatus
  currentError: string | null
  load: () => Promise<void>
  loadOne: (id: string) => Promise<void>
  create: (body: {
    applicationId: string
    title: string
    kind: PushDocument['kind']
    template: PushDocument['template']
    language?: string
  }) => Promise<PushDocument>
  resetCurrent: () => void
}

export const useDocumentsStore = create<DocumentsState>()((set) => ({
  items: [],
  status: 'idle',
  error: null,
  current: null,
  currentVersion: null,
  currentStatus: 'idle',
  currentError: null,
  load: async () => {
    set({ status: 'loading', error: null })
    try {
      const env = await listDocuments({ limit: 50 })
      set({ items: env.data, status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  loadOne: async (id) => {
    set({ currentStatus: 'loading', currentError: null })
    try {
      const document = await getDocument(id)
      let version: DocumentVersion | null = null
      if (document.latestVersionId) {
        version = await getDocumentVersion(id, document.latestVersionId)
      }
      set({
        current: document,
        currentVersion: version,
        currentStatus: 'success',
      })
    } catch (error) {
      set({
        currentStatus: 'error',
        currentError: error instanceof Error ? error.message : 'error',
      })
    }
  },
  create: async (body) => {
    const created = await createDocument(body)
    set((s) => ({ items: [created, ...s.items] }))
    return created
  },
  resetCurrent: () =>
    set({
      current: null,
      currentVersion: null,
      currentStatus: 'idle',
      currentError: null,
    }),
}))
