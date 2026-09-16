import { useEffect } from 'react'
import {
  Button,
  CanvasHeader,
  ErrorState,
  Form,
  FormRow,
  FormSection,
  Select,
  SkeletonRows,
  showToast,
} from '@/shared/ui'
import { persistTheme, useSettingsStore } from '../model/settings-store'

const SettingsPage = () => {
  const me = useSettingsStore((s) => s.me)
  const billing = useSettingsStore((s) => s.billing)
  const models = useSettingsStore((s) => s.models)
  const status = useSettingsStore((s) => s.status)
  const error = useSettingsStore((s) => s.error)
  const theme = useSettingsStore((s) => s.theme)
  const setTheme = useSettingsStore((s) => s.setTheme)
  const load = useSettingsStore((s) => s.load)

  useEffect(() => {
    void load()
  }, [load])

  const save = () => {
    persistTheme(theme)
    showToast('저장했어요', 'check')
  }

  const modelLabel = models.find((m) => m.available)?.label ?? '—'

  return (
    <>
      <CanvasHeader title="설정" />
      <div className="canvas-body">
        {status === 'loading' ? <SkeletonRows count={6} height={40} /> : null}
        {status === 'error' ? (
          <ErrorState message={error ?? undefined} onRetry={() => void load()} />
        ) : null}
        {status === 'success' ? (
          <Form>
            <FormSection title="계정">
              <FormRow label="이름" hint={me?.displayName ?? '—'} />
              <FormRow label="로캘" hint={me?.locale ?? '—'} />
            </FormSection>
            <FormSection title="AI">
              <FormRow label="모델" hint={modelLabel} />
            </FormSection>
            <FormSection title="요금제">
              <FormRow label="현재 플랜" hint={billing?.plan ?? 'Free'} />
            </FormSection>
            <FormSection title="외관">
              <FormRow label="테마">
                <Select
                  aria-label="테마"
                  value={theme}
                  onChange={(e) => setTheme(e.target.value)}
                >
                  <option value="라이트">라이트</option>
                </Select>
              </FormRow>
            </FormSection>
            <Button variant="primary" size="md" onClick={save}>
              저장
            </Button>
          </Form>
        ) : null}
      </div>
    </>
  )
}

export default SettingsPage
