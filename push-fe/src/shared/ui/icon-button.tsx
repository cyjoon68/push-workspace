import type { ButtonHTMLAttributes } from 'react'
import Icon from './icon'
import type { IconName } from './icon'

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  icon: IconName
  iconSize?: 16 | 20 | 24
  'aria-label': string
}

const IconButton = ({
  icon,
  iconSize = 20,
  className,
  ...rest
}: IconButtonProps) => (
  <button
    type="button"
    className={['icon-button', className ?? ''].filter(Boolean).join(' ')}
    {...rest}
  >
    <Icon name={icon} size={iconSize} />
  </button>
)

export default IconButton
