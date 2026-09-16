import { useEffect, useState } from 'react'
import { Button, Dialog, Input, Select, showToast } from '@/shared/ui'
import { useApplicationsStore } from '@/entities/application'
import { useInterviewsStore } from '../model/interviews-store'

type InterviewAddDialogProps = {
  open: boolean
  onClose: () => void
}

const InterviewAddDialog = ({ open, onClose }: InterviewAddDialogProps) => {
  const applications = useApplicationsStore((s) => s.items)
  const loadApplications = useApplicationsStore((s) => s.load)
  const create = useInterviewsStore((s) => s.create)
  const [applicationId, setApplicationId] = useState('')
  const [title, setTitle] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (open && applications.length === 0) void loadApplications()
  }, [open, applications.length, loadApplications])

  const submit = async () => {
    if (!applicationId || !title.trim() || !scheduledAt) return
    setBusy(true)
    try {
      await create({
        applicationId,
        title: title.trim(),
        scheduledAt: new Date(scheduledAt).toISOString(),
      })
      showToast('면접 준비를 만들었어요', 'check')
      onClose()
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '만들지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      title="면접 준비"
      onClose={onClose}
      actions={
        <>
          <Button variant="secondary" size="sm" onClick={onClose}>
            취소
          </Button>
          <Button
            variant="primary"
            size="sm"
            loading={busy}
            disabled={!applicationId || !title.trim() || !scheduledAt}
            onClick={() => void submit()}
          >
            추가
          </Button>
        </>
      }
    >
      <div className="form" style={{ gap: 'var(--space-12)' }}>
        <Select
          aria-label="연결할 지원"
          value={applicationId}
          onChange={(e) => setApplicationId(e.target.value)}
        >
          <option value="">지원 선택</option>
          {applications.map((a) => (
            <option key={a.id} value={a.id}>
              {a.company} {a.title}
            </option>
          ))}
        </Select>
        <Input
          placeholder="면접 이름"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Input
          type="datetime-local"
          aria-label="면접 일시"
          value={scheduledAt}
          onChange={(e) => setScheduledAt(e.target.value)}
        />
        {applications.length === 0 ? (
          <span className="form-row-hint">
            먼저 지원 관리에서 공고를 추가해 주세요.
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

export default InterviewAddDialog
