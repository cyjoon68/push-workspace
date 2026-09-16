import ky from 'ky'
import { API_BASE_URL } from '../constants'
import { getAccessToken } from '../auth/session'

export const api = ky.create({
  prefixUrl: API_BASE_URL,
  timeout: 15_000,
  retry: {
    limit: 1,
    methods: ['get'],
    statusCodes: [408, 502, 503, 504],
  },
  hooks: {
    beforeRequest: [
      (req) => {
        const token = getAccessToken()
        if (token) req.headers.set('Authorization', `Bearer ${token}`)
      },
    ],
  },
})
