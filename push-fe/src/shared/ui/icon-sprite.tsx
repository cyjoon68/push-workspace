import spriteRaw from '../design/icons.svg?raw'

const IconSprite = () => (
  <div
    style={{ display: 'none' }}
    aria-hidden="true"
    dangerouslySetInnerHTML={{ __html: spriteRaw }}
  />
)

export default IconSprite
