type SuggestChipsProps = {
  items: readonly string[]
  onSelect?: (item: string) => void
}

const SuggestChips = ({ items, onSelect }: SuggestChipsProps) => (
  <div className="suggest-chips">
    {items.map((item) => (
      <button
        key={item}
        type="button"
        className="suggest-chips-item"
        onClick={() => onSelect?.(item)}
      >
        {item}
      </button>
    ))}
  </div>
)

export default SuggestChips
