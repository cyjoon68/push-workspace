import Icon from './icon'
import Button from './button'

type EmptyStateProps = {
  message: string
  actionLabel?: string
  onAction?: () => void
}

const EmptyState = ({ message, actionLabel, onAction }: EmptyStateProps) => (
  <div className="empty-state">
    <div className="empty-state-icon">
      <Icon name="inbox" size={48} />
    </div>
    <p className="t-body">{message}</p>
    {actionLabel ? (
      <Button variant="secondary" size="md" onClick={onAction}>
        {actionLabel}
      </Button>
    ) : null}
  </div>
)

export default EmptyState
