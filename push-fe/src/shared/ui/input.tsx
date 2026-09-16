import type { InputHTMLAttributes, TextareaHTMLAttributes } from 'react'

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  error?: boolean
}

const Input = ({ error, className, ...rest }: InputProps) => (
  <input
    className={['input', error ? 'is-error' : '', className ?? '']
      .filter(Boolean)
      .join(' ')}
    {...rest}
  />
)

const Textarea = (props: TextareaHTMLAttributes<HTMLTextAreaElement>) => (
  <textarea className="input input-textarea" {...props} />
)

const Select = (props: React.SelectHTMLAttributes<HTMLSelectElement>) => (
  <select className="input" {...props} />
)

export { Input, Textarea, Select }
