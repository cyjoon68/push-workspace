import { useEffect } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { PROJECT_TAG_LABEL } from '@/shared/constants'
import { Card, CanvasHeader, IconButton, StatusChip } from '@/shared/ui'
import type { StatusChipTone } from '@/shared/ui'
import type { CliRunState } from '@/shared/api'
import {
  isProjectConversation,
  useConversationsStore,
} from '@/entities/conversation'
import { ChatStream, Composer, useMessagesStore } from '@/features/chat'
import { useProjectPanelsStore } from '../model/project-panels'

const RUN_TONES: Record<CliRunState, StatusChipTone> = {
  DRAFT: 'pending',
  APPROVAL_REQUIRED: 'pending',
  RUNNING: 'running',
  VERIFYING: 'running',
  VERIFIED: 'verified',
  FAILED: 'error',
}

const ProjectPanels = ({ projectId }: { projectId: string }) => {
  const runs = useProjectPanelsStore((s) => s.runs)
  const evidence = useProjectPanelsStore((s) => s.evidence)
  const load = useProjectPanelsStore((s) => s.load)

  useEffect(() => {
    void load(projectId)
  }, [projectId, load])

  return (
    <>
      {runs.map((run) => (
        <Card key={run.id} title="실행 상태">
          <StatusChip
            tone={RUN_TONES[run.state]}
            label={run.state}
            icon={run.state === 'RUNNING' ? 'loader-circle' : undefined}
          />
          <div className="card-meta">
            {run.executable}
            {run.arguments.length ? ` ${run.arguments.join(' ')}` : ''}
          </div>
        </Card>
      ))}
      {evidence.map((item) => (
        <Card key={item.id} title="검증 결과">
          <StatusChip
            tone={
              item.status === 'VERIFIED'
                ? 'verified'
                : item.status === 'REJECTED'
                  ? 'error'
                  : 'pending'
            }
            label={item.status}
            icon={item.status === 'VERIFIED' ? 'badge-check' : undefined}
          />
          {item.summary ? (
            <div className="card-meta">{item.summary}</div>
          ) : null}
        </Card>
      ))}
    </>
  )
}

const ChatPage = () => {
  const { id = '' } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  const conversation = useConversationsStore((s) =>
    s.items.find((c) => c.id === id),
  )
  const messages = useMessagesStore((s) => s.messages)
  const status = useMessagesStore((s) => s.status)
  const error = useMessagesStore((s) => s.error)
  const sending = useMessagesStore((s) => s.sending)
  const load = useMessagesStore((s) => s.load)
  const send = useMessagesStore((s) => s.send)
  const loadConversations = useConversationsStore((s) => s.load)

  useEffect(() => {
    void loadConversations()
  }, [loadConversations])

  useEffect(() => {
    void load(id)
  }, [id, load])

  const initialMessage = (
    location.state as { initialMessage?: string } | null
  )?.initialMessage

  useEffect(() => {
    if (!initialMessage || status !== 'success') return
    void send(id, initialMessage)
    navigate(location.pathname, { replace: true, state: null })
  }, [initialMessage, status, id, send, navigate, location.pathname])

  const isProject = conversation ? isProjectConversation(conversation) : false
  const projectId = conversation?.projectId ?? null

  return (
    <>
      <CanvasHeader
        title={conversation?.title ?? '새 채팅'}
        actions={
          <>
            {isProject ? (
              <StatusChip tone="ready" label={PROJECT_TAG_LABEL} />
            ) : null}
            <IconButton icon="ellipsis" aria-label="더보기" />
          </>
        }
      />
      <ChatStream
        messages={messages}
        status={status}
        error={error}
        onRetry={() => void load(id)}
        onSelectSuggestion={(text) => void send(id, text)}
        trailing={
          isProject && projectId ? (
            <ProjectPanels projectId={projectId} />
          ) : null
        }
      />
      <Composer
        sending={sending}
        onSend={(text) => void send(id, text)}
      />
    </>
  )
}

export default ChatPage
