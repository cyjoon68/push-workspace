import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { PROJECT_TAG_LABEL, ROUTES } from '@/shared/constants'
import { Icon, IconButton, showToast } from '@/shared/ui'
import type { IconName } from '@/shared/ui'
import {
  isProjectConversation,
  useConversationsStore,
} from '@/entities/conversation'
import { useMessagesStore } from '@/features/chat'

const NAV_ITEMS: { to: string; icon: IconName; label: string }[] = [
  { to: ROUTES.documents, icon: 'file-text', label: '내 서류' },
  { to: ROUTES.applications, icon: 'briefcase', label: '지원 관리' },
  { to: ROUTES.vault, icon: 'archive', label: '커리어 볼트' },
  { to: ROUTES.interview, icon: 'mic', label: '면접' },
  { to: ROUTES.calendar, icon: 'calendar', label: '캘린더' },
]

const Sidebar = () => {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const conversations = useConversationsStore((s) => s.items)
  const loadConversations = useConversationsStore((s) => s.load)
  const createConversation = useConversationsStore((s) => s.create)
  const sendingTo = useMessagesStore((s) =>
    s.sending ? s.conversationId : null,
  )

  useEffect(() => {
    void loadConversations()
  }, [loadConversations])

  const activeChatId = location.pathname.startsWith('/chat/')
    ? location.pathname.split('/')[2]
    : null

  const newChat = async () => {
    try {
      const conversation = await createConversation({
        applicationId: null,
        title: '새 채팅',
      })
      navigate(ROUTES.chat(conversation.id))
    } catch (error) {
      showToast(
        error instanceof Error ? error.message : '채팅을 만들지 못했어요',
        'circle-alert',
      )
    }
  }

  const dotClass = (id: string) => {
    if (sendingTo === id) return 'sidebar-dot is-busy'
    if (id === activeChatId) return 'sidebar-dot is-on'
    return 'sidebar-dot'
  }

  return (
    <aside className={collapsed ? 'sidebar is-collapsed' : 'sidebar'}>
      <div className="sidebar-head">
        <span className="t-label">Push</span>
        <IconButton
          icon={collapsed ? 'panel-left-open' : 'panel-left-close'}
          aria-label={collapsed ? '사이드바 펼치기' : '사이드바 접기'}
          onClick={() => setCollapsed((v) => !v)}
        />
      </div>
      <div className="sidebar-section">
        <div className="sidebar-label">채팅</div>
        <button
          type="button"
          className="sidebar-item"
          onClick={() => void newChat()}
        >
          <Icon name="square-pen" size={20} />
          <span className="sidebar-item-label">새 채팅</span>
        </button>
        {conversations.map((c) => (
          <button
            key={c.id}
            type="button"
            className={
              c.id === activeChatId ? 'sidebar-item is-active' : 'sidebar-item'
            }
            onClick={() => navigate(ROUTES.chat(c.id))}
          >
            <span className={dotClass(c.id)} />
            <span className="sidebar-item-label">{c.title}</span>
            {isProjectConversation(c) ? (
              <span className="sidebar-tag">{PROJECT_TAG_LABEL}</span>
            ) : null}
          </button>
        ))}
      </div>
      <div className="sidebar-section">
        <div className="sidebar-label">기능</div>
        {NAV_ITEMS.map((item) => (
          <button
            key={item.to}
            type="button"
            className={
              location.pathname.startsWith(item.to)
                ? 'sidebar-item is-active'
                : 'sidebar-item'
            }
            onClick={() => navigate(item.to)}
          >
            <Icon name={item.icon} size={20} />
            <span className="sidebar-item-label">{item.label}</span>
          </button>
        ))}
      </div>
      <div className="sidebar-foot">
        <button
          type="button"
          className={
            location.pathname === ROUTES.settings
              ? 'sidebar-item is-active'
              : 'sidebar-item'
          }
          onClick={() => navigate(ROUTES.settings)}
        >
          <Icon name="settings" size={20} />
          <span className="sidebar-item-label">설정</span>
        </button>
      </div>
    </aside>
  )
}

export default Sidebar
