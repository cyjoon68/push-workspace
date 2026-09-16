import type { ReactNode } from 'react'
import type { Message, MessageAttachment } from '@/shared/api'
import { ApprovalCard } from '@/features/approval'
import {
  Card,
  ErrorState,
  Icon,
  SkeletonRows,
  SuggestChips,
} from '@/shared/ui'
import { HOME_SUGGESTIONS } from '@/shared/constants'

const AttachmentView = ({ attachment }: { attachment: MessageAttachment }) => {
  if (attachment.type === 'APPROVAL') {
    return <ApprovalCard approvalId={attachment.id} />
  }
  if (attachment.type === 'EVIDENCE') {
    return (
      <Card title={attachment.title}>
        <span className="card-meta">
          <Icon name="link-2" size={16} /> 근거 연결됨
        </span>
      </Card>
    )
  }
  return (
    <Card title={attachment.title}>
      <span className="card-meta">문서 버전</span>
    </Card>
  )
}

const MessageView = ({ message }: { message: Message }) => (
  <>
    <div
      className={[
        'chat-stream-msg',
        message.role === 'USER'
          ? 'chat-stream-msg-user'
          : 'chat-stream-msg-ai',
      ].join(' ')}
    >
      {message.text}
    </div>
    {message.attachments.map((a, i) => (
      <AttachmentView key={`${message.id}-${i}`} attachment={a} />
    ))}
  </>
)

type ChatStreamProps = {
  messages: Message[]
  status: 'idle' | 'loading' | 'success' | 'error'
  error?: string | null
  onRetry?: () => void
  onSelectSuggestion?: (text: string) => void
  trailing?: ReactNode
}

const ChatStream = ({
  messages,
  status,
  error,
  onRetry,
  onSelectSuggestion,
  trailing,
}: ChatStreamProps) => {
  if (status === 'loading') {
    return (
      <div className="chat-stream">
        <SkeletonRows count={4} />
      </div>
    )
  }
  if (status === 'error') {
    return (
      <div className="canvas-body">
        <ErrorState message={error ?? undefined} onRetry={onRetry} />
      </div>
    )
  }
  return (
    <div className="chat-stream">
      {messages.length === 0 ? (
        <SuggestChips items={HOME_SUGGESTIONS} onSelect={onSelectSuggestion} />
      ) : (
        messages.map((m) => <MessageView key={m.id} message={m} />)
      )}
      {trailing}
    </div>
  )
}

export default ChatStream
