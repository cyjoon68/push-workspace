import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { HOME_SUGGESTIONS, ROUTES } from '@/shared/constants'
import { isHttpUrl } from '@/shared/lib/url'
import {
  Card,
  CardGrid,
  ErrorState,
  SkeletonCardGrid,
  SuggestChips,
  showToast,
} from '@/shared/ui'
import { StatusChip } from '@/shared/ui'
import { stageChipTone, useApplicationsStore } from '@/entities/application'
import { useConversationsStore } from '@/entities/conversation'
import { JobAddDialog } from '@/features/job-add'
import CommandInput from './command-input'

const RECENT_LIMIT = 3

const HomePage = () => {
  const navigate = useNavigate()
  const applications = useApplicationsStore((s) => s.items)
  const status = useApplicationsStore((s) => s.status)
  const error = useApplicationsStore((s) => s.error)
  const load = useApplicationsStore((s) => s.load)
  const createConversation = useConversationsStore((s) => s.create)
  const [busy, setBusy] = useState(false)
  const [jobDialog, setJobDialog] = useState<{ open: boolean; source: string }>({
    open: false,
    source: '',
  })

  useEffect(() => {
    if (status === 'idle') void load()
  }, [status, load])

  const submit = async (text: string) => {
    if (isHttpUrl(text)) {
      setJobDialog({ open: true, source: text })
      return
    }
    setBusy(true)
    try {
      const conversation = await createConversation({
        applicationId: null,
        title: text.slice(0, 40),
      })
      navigate(ROUTES.chat(conversation.id), {
        state: { initialMessage: text },
      })
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '실행하지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  const recents = applications.slice(0, RECENT_LIMIT)

  return (
    <>
      <div className="canvas-body">
        {status === 'error' ? (
          <ErrorState message={error ?? undefined} onRetry={() => void load()} />
        ) : (
          <>
            <div className="center">
              <h1 className="t-display">무엇을 도와드릴까요?</h1>
              <CommandInput onSubmit={(t) => void submit(t)} busy={busy} />
              <SuggestChips
                items={HOME_SUGGESTIONS}
                onSelect={(t) => void submit(t)}
              />
            </div>
            {status === 'loading' ? (
              <>
                <div className="canvas-header">
                  <h2 className="canvas-header-title">최근 작업</h2>
                </div>
                <SkeletonCardGrid count={3} />
              </>
            ) : null}
            {status === 'success' && recents.length > 0 ? (
              <>
                <div className="canvas-header">
                  <h2 className="canvas-header-title">최근 작업</h2>
                </div>
                <CardGrid>
                  {recents.map((a) => (
                    <Card
                      key={a.id}
                      title={`${a.company} ${a.title}`}
                      meta={a.notes || undefined}
                      onClick={() => navigate(ROUTES.job(a.jobId))}
                    >
                      <StatusChip
                        tone={stageChipTone(a.stage)}
                        label={a.stage}
                      />
                    </Card>
                  ))}
                </CardGrid>
              </>
            ) : null}
          </>
        )}
      </div>
      <JobAddDialog
        open={jobDialog.open}
        initialSource={jobDialog.source}
        onClose={() => setJobDialog({ open: false, source: '' })}
        onCreated={() => void load()}
      />
    </>
  )
}

export default HomePage
