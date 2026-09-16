import { useEffect } from 'react'
import { useInferenceSettings } from '../model/inference-settings'

const EFFORT_LABELS = [
  { value: 'LOW', label: '낮음' },
  { value: 'MEDIUM', label: '보통' },
  { value: 'HIGH', label: '높음' },
] as const

const InferenceSettings = () => {
  const effort = useInferenceSettings((s) => s.effort)
  const ultraResume = useInferenceSettings((s) => s.ultraResume)
  const models = useInferenceSettings((s) => s.models)
  const modelsStatus = useInferenceSettings((s) => s.modelsStatus)
  const setEffort = useInferenceSettings((s) => s.setEffort)
  const setUltraResume = useInferenceSettings((s) => s.setUltraResume)
  const loadModels = useInferenceSettings((s) => s.loadModels)

  useEffect(() => {
    if (modelsStatus === 'idle') void loadModels()
  }, [modelsStatus, loadModels])

  const activeModel = models.find((m) => m.available)

  return (
    <div className="inference-popover" role="dialog" aria-label="추론 설정">
      <div className="form-section">
        <div className="form-section-title">추론 설정</div>
        <div className="form-row">
          <span className="form-row-label">모델</span>
          <span className="form-row-hint">
            {modelsStatus === 'success'
              ? (activeModel?.label ?? '사용 가능한 모델 없음')
              : 'OpenAI'}
          </span>
        </div>
        <div className="form-row">
          <span className="form-row-label">추론 강도</span>
          <div className="inference-effort">
            {EFFORT_LABELS.map((o) => (
              <button
                key={o.value}
                type="button"
                className={[
                  'button',
                  'button-sm',
                  effort === o.value
                    ? 'is-selected'
                    : 'button-secondary',
                ].join(' ')}
                onClick={() => setEffort(o.value)}
              >
                {o.label}
              </button>
            ))}
          </div>
        </div>
        <div className="form-row">
          <span className="form-row-label">UltraResume</span>
          <button
            type="button"
            role="switch"
            aria-checked={ultraResume}
            aria-label="UltraResume"
            className={[
              'switch',
              ultraResume ? 'is-on' : '',
            ]
              .filter(Boolean)
              .join(' ')}
            onClick={() => setUltraResume(!ultraResume)}
          >
            <span className="switch-thumb" />
          </button>
        </div>
      </div>
    </div>
  )
}

export default InferenceSettings
