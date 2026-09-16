type DocCardProps = {
  title: string
  meta: string
  onOpen: () => void
}

const DocCard = ({ title, meta, onOpen }: DocCardProps) => (
  <button type="button" className="doc-card" onClick={onOpen}>
    <span className="doc-card-title">{title}</span>
    <span className="doc-card-meta">{meta}</span>
  </button>
)

export default DocCard
