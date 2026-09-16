import Icon from './icon'
import type { IconName } from './icon'

export type StatusChipTone =
  | 'ready'
  | 'running'
  | 'verified'
  | 'pending'
  | 'error'

type StatusChipProps = {
  tone: StatusChipTone
  label: string
  icon?: IconName
}

const StatusChip = ({ tone, label, icon }: StatusChipProps) => (
  <span className={`status-chip status-chip-${tone}`}>
    {icon ? <Icon name={icon} size={16} /> : null}
    {label}
  </span>
)

export default StatusChip
