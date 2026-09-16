import { create } from 'zustand'
import type { Message } from '@/shared/api'
import { listMessages, sendMessage, waitForOperation } from '@/shared/api'
import { useApprovalsStore } from '@/entities/approval'
import { useInferenceSettings } from './inference-settings'
import { showToast } from '@/shared/ui'

type LoadStatus = 'idle' | 'loading' | 'success' | 'error'

type ChatMessageResult = {
  userMessage: Message
  assistantMessage: Message
  approvalIds: string[]
}

const isChatMessageResult = (
  result: unknown,
): result is { kind: 'CHAT_MESSAGE'; value: ChatMessageResult } =>
  typeof result === 'object' &&
  result !== null &&
  (result as { kind?: string }).kind === 'CHAT_MESSAGE'

type MessagesState = {
  conversationId: string | null
  messages: Message[]
  status: LoadStatus
  error: string | null
  sending: boolean
  load: (conversationId: string) => Promise<void>
  send: (conversationId: string, text: string) => Promise<void>
  reset: () => void
}

export const useMessagesStore = create<MessagesState>()((set) => ({
  conversationId: null,
  messages: [],
  status: 'idle',
  error: null,
  sending: false,
  load: async (conversationId) => {
    set({ conversationId, status: 'loading', error: null, messages: [] })
    try {
      const env = await listMessages(conversationId, { limit: 50 })
      set({ messages: [...env.data].reverse(), status: 'success' })
    } catch (error) {
      set({
        status: 'error',
        error: error instanceof Error ? error.message : 'error',
      })
    }
  },
  send: async (conversationId, text) => {
    const ai = useInferenceSettings.getState().aiOptions()
    if (!ai) {
      showToast('AI 모델 설정을 불러오지 못했어요', 'circle-alert')
      return
    }
    const accessMode = useInferenceSettings.getState().accessMode
    const tempId = `local-${crypto.randomUUID()}`
    const optimistic: Message = {
      id: tempId,
      conversationId,
      role: 'USER',
      text,
      attachments: [],
      operationId: null,
      createdAt: new Date().toISOString(),
    }
    set((s) => ({ messages: [...s.messages, optimistic], sending: true }))
    try {
      const operation = await sendMessage(conversationId, {
        text,
        context: { evidenceIds: [] },
        ai,
        accessMode,
      })
      const done = await waitForOperation(operation.id)
      const result = done.result
      if (done.status === 'SUCCEEDED' && isChatMessageResult(result)) {
        const { userMessage, assistantMessage, approvalIds } = result.value
        set((s) => ({
          messages: [
            ...s.messages.filter((m) => m.id !== tempId),
            userMessage,
            assistantMessage,
          ],
          sending: false,
        }))
        for (const id of approvalIds) {
          void useApprovalsStore.getState().ensure(id)
        }
      } else {
        set((s) => ({
          messages: s.messages.filter((m) => m.id !== tempId),
          sending: false,
        }))
        showToast(
          done.error?.message ?? '응답을 받지 못했어요',
          'circle-alert',
        )
      }
    } catch (error) {
      set((s) => ({
        messages: s.messages.filter((m) => m.id !== tempId),
        sending: false,
      }))
      showToast(
        error instanceof Error ? error.message : '전송하지 못했어요',
        'circle-alert',
      )
    }
  },
  reset: () =>
    set({
      conversationId: null,
      messages: [],
      status: 'idle',
      error: null,
      sending: false,
    }),
}))
