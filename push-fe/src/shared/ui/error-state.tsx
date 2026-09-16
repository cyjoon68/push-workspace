import Button from './button'
import { NETWORK_ERROR_MESSAGE } from '../api/envelope'

type ErrorStateProps = {
  title?: string
  message?: string
  onRetry?: () => void
}

const ErrorState = ({
  title = '불러오지 못했어요',
  message = NETWORK_ERROR_MESSAGE,
  onRetry,
}: ErrorStateProps) => (
  <div className="error-state">
    <div className="t-h3">{title}</div>
    <p className="t-body-sm">{message}</p>
    {onRetry ? (
      <Button variant="secondary" size="md" onClick={onRetry}>
        다시 시도
      </Button>
    ) : null}
  </div>
)

export default ErrorState
