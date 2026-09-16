import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/shared/constants'
import type { DocumentKind } from '@/shared/api'
import { Button, Dialog, Input, Select, showToast } from '@/shared/ui'
import { useApplicationsStore } from '@/entities/application'
import { useDocumentsStore } from '@/entities/document'

const KIND_OPTIONS: { value: DocumentKind; label: string }[] = [
  { value: 'RESUME', label: '이력서' },
  { value: 'COVER_LETTER', label: '자기소개서' },
  { value: 'PORTFOLIO', label: '포트폴리오' },
]

type NewDocumentDialogProps = {
  open: boolean
  onClose: () => void
}

const NewDocumentDialog = ({ open, onClose }: NewDocumentDialogProps) => {
  const navigate = useNavigate()
  const applications = useApplicationsStore((s) => s.items)
  const loadApplications = useApplicationsStore((s) => s.load)
  const create = useDocumentsStore((s) => s.create)
  const [title, setTitle] = useState('')
  const [kind, setKind] = useState<DocumentKind>('RESUME')
  const [applicationId, setApplicationId] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (open && applications.length === 0) void loadApplications()
  }, [open, applications.length, loadApplications])

  const submit = async () => {
    if (!title.trim() || !applicationId) return
    setBusy(true)
    try {
      const doc = await create({
        applicationId,
        title: title.trim(),
        kind,
        template: 'CLASSIC',
      })
      onClose()
      navigate(ROUTES.document(doc.id))
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '문서를 만들지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      title="새 문서"
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
            disabled={!title.trim() || !applicationId}
            onClick={() => void submit()}
          >
            만들기
          </Button>
        </>
      }
    >
      <div className="form" style={{ gap: 'var(--space-12)' }}>
        <Input
          placeholder="문서 제목"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Select
          aria-label="문서 종류"
          value={kind}
          onChange={(e) => setKind(e.target.value as DocumentKind)}
        >
          {KIND_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
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
        {applications.length === 0 ? (
          <span className="form-row-hint">
            먼저 지원 관리에서 공고를 추가해 주세요.
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

export default NewDocumentDialog
