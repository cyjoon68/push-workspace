export type IconName =
  | 'search'
  | 'panel-left-close'
  | 'panel-left-open'
  | 'square-pen'
  | 'file-text'
  | 'briefcase'
  | 'archive'
  | 'mic'
  | 'calendar'
  | 'settings'
  | 'corner-down-left'
  | 'send-horizontal'
  | 'sliders-horizontal'
  | 'paperclip'
  | 'link-2'
  | 'badge-check'
  | 'loader-circle'
  | 'check'
  | 'x'
  | 'ellipsis'
  | 'trash-2'
  | 'file-down'
  | 'pencil'
  | 'plus'
  | 'circle-alert'
  | 'inbox'

const STROKE_WIDTHS: Record<number, number> = { 16: 1.5, 20: 1.75, 24: 2 }

type IconProps = {
  name: IconName
  size?: 16 | 20 | 24 | 48
}

const Icon = ({ name, size = 20 }: IconProps) => (
  <svg
    className="icon"
    width={size}
    height={size}
    strokeWidth={STROKE_WIDTHS[size] ?? 1.75}
    aria-hidden="true"
  >
    <use href={`#i-${name}`} />
  </svg>
)

export default Icon
