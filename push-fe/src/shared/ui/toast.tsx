import { useEffect } from 'react'
import { create } from 'zustand'
import Icon from './icon'
import type { IconName } from './icon'

type ToastItem = {
  id: number
  message: string
  icon?: IconName
}

type ToastState = {
  toasts: ToastItem[]
  show: (message: string, icon?: IconName) => void
  dismiss: (id: number) => void
}

let nextId = 1

export const useToastStore = create<ToastState>()((set) => ({
  toasts: [],
  show: (message, icon) =>
    set((s) => ({ toasts: [...s.toasts, { id: nextId++, message, icon }] })),
  dismiss: (id) =>
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))

export const showToast = (message: string, icon: IconName = 'check'): void =>
  useToastStore.getState().show(message, icon)

const ToastItemView = ({ toast }: { toast: ToastItem }) => {
  const dismiss = useToastStore((s) => s.dismiss)
  useEffect(() => {
    const timer = setTimeout(() => dismiss(toast.id), 4_000)
    return () => clearTimeout(timer)
  }, [toast.id, dismiss])
  return (
    <div className="toast" role="status">
      {toast.icon ? <Icon name={toast.icon} size={16} /> : null}
      {toast.message}
    </div>
  )
}

const ToastHost = () => {
  const toasts = useToastStore((s) => s.toasts)
  return (
    <>
      {toasts.map((t) => (
        <ToastItemView key={t.id} toast={t} />
      ))}
    </>
  )
}

export default ToastHost
