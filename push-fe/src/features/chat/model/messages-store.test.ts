import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Message } from '@/shared/api'

vi.mock('@/shared/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/shared/api')>()
  return {
    ...mod,
    listMessages: vi.fn(),
    sendMessage: vi.fn(),
    waitForOperation: vi.fn(),
  }
})

import { listMessages, sendMessage } from '@/shared/api'
import { useInferenceSettings } from './inference-settings'
import { useMessagesStore } from './messages-store'

const message = (over: Partial<Message> = {}): Message => ({
  id: 'm1',
  conversationId: 'c1',
  role: 'ASSISTANT',
  text: 'hi',
  attachments: [],
  operationId: null,
  createdAt: '2026-09-01T00:00:00Z',
  ...over,
})

describe('messages store', () => {
  beforeEach(() => {
    vi.mocked(listMessages).mockReset()
    vi.mocked(sendMessage).mockReset()
    useMessagesStore.getState().reset()
    useInferenceSettings.setState({ models: [], modelsStatus: 'idle' })
  })

  it('loads messages for a conversation', async () => {
    vi.mocked(listMessages).mockResolvedValue({
      data: [message()],
      page: { nextCursor: null, hasMore: false },
    })
    await useMessagesStore.getState().load('c1')
    const s = useMessagesStore.getState()
    expect(s.status).toBe('success')
    expect(s.conversationId).toBe('c1')
    expect(s.messages).toHaveLength(1)
  })

  it('surfaces load failures as error status', async () => {
    vi.mocked(listMessages).mockRejectedValue(new Error('down'))
    await useMessagesStore.getState().load('c1')
    expect(useMessagesStore.getState().status).toBe('error')
  })

  it('never calls the API when no AI model is configured', async () => {
    await useMessagesStore.getState().send('c1', 'hello')
    expect(sendMessage).not.toHaveBeenCalled()
    expect(useMessagesStore.getState().sending).toBe(false)
    expect(useMessagesStore.getState().messages).toHaveLength(0)
  })
})
