export const API_BASE_URL = 'http://localhost:8080/api/v1'

export const ROUTES = {
  login: '/login',
  home: '/',
  chat: (id: string) => `/chat/${id}`,
  documents: '/documents',
  document: (id: string) => `/documents/${id}`,
  applications: '/applications',
  job: (id: string) => `/jobs/${id}`,
  vault: '/vault',
  interview: '/interview',
  calendar: '/calendar',
  settings: '/settings',
} as const

export const HOME_SUGGESTIONS = [
  '지원 현황 정리해줘',
  '이력서 초안 써줘',
  '면접 질문 뽑아줘',
] as const

export const PROJECT_TAG_LABEL = '프로젝트'

export const COMPOSER_PLACEHOLDER =
  '메시지 입력 — UltraResume 등 추론 설정은 슬라이더 아이콘에서'

export const COMMAND_INPUT_PLACEHOLDER =
  '공고 URL을 붙여넣거나, 명령을 입력하세요'
