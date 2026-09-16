import { useState } from 'react'
import type { EvidenceKind } from '@/shared/api'
import { Button, Dialog, Input, Select, Textarea, showToast } from '@/shared/ui'
import { useEvidenceStore } from '@/entities/evidence'

const KIND_OPTIONS: { value: EvidenceKind; label: string }[] = [
  { value: 'CAREER', label: '경력' },
  { value: 'PROJECT', label: '프로젝트' },
  { value: 'RESUME', label: '이력서' },
  { value: 'GITHUB', label: 'GitHub' },
  { value: 'EDUCATION', label: '학력' },
  { value: 'SKILL', label: '기술' },
]

type EvidenceAddDialogProps = {
  open: boolean
  onClose: () => void
}

const EvidenceAddDialog = ({ open, onClose }: EvidenceAddDialogProps) => {
  const create = useEvidenceStore((s) => s.create)
  const [kind, setKind] = useState<EvidenceKind>('CAREER')
  const [title, setTitle] = useState('')
  const [sourceText, setSourceText] = useState('')
  const [sourceUrl, setSourceUrl] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (!title.trim() || !sourceText.trim()) return
    setBusy(true)
    try {
      await create({
        kind,
        title: title.trim(),
        sourceText: sourceText.trim(),
        ...(sourceUrl.trim() ? { sourceUrl: sourceUrl.trim() } : {}),
      })
      showToast('근거를 추가했어요', 'check')
      onClose()
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '추가하지 못했어요',
        'circle-alert',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      title="근거 추가"
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
            disabled={!title.trim() || !sourceText.trim()}
            onClick={() => void submit()}
          >
            추가
          </Button>
        </>
      }
    >
      <div className="form" style={{ gap: 'var(--space-12)' }}>
        <Select
          aria-label="근거 종류"
          value={kind}
          onChange={(e) => setKind(e.target.value as EvidenceKind)}
        >
          {KIND_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
        <Input
          placeholder="제목"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Textarea
          placeholder="원문을 붙여넣으세요 (수치·경험은 원문 그대로)"
          value={sourceText}
          onChange={(e) => setSourceText(e.target.value)}
          rows={5}
        />
        <Input
          placeholder="출처 URL (선택)"
          value={sourceUrl}
          onChange={(e) => setSourceUrl(e.target.value)}
        />
      </div>
    </Dialog>
  )
}

export default EvidenceAddDialog
