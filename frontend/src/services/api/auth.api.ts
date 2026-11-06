// Endpoint /auth/* core-api (ADR 0002).
import { core } from '@/services/http/client'
import type { Profile, Session } from '@/types/auth.types'

export const authApi = {
  login: (email: string, password: string) => core.post<Session>('/auth/login', { email, password }, { anonymous: true }),
  refresh: (refresh_token: string) => core.post<Session>('/auth/refresh', { refresh_token }, { anonymous: true }),
  switchProject: (refresh_token: string, project_id: string) =>
    core.post<Session>('/auth/switch-project', { refresh_token, project_id }, { anonymous: true }),
  logout: (refresh_token: string) => core.post<void>('/auth/logout', { refresh_token }, { anonymous: true }),
  me: () => core.get<Profile>('/auth/me'),
  changePassword: (current_password: string, new_password: string) => core.post('/auth/change-password', { current_password, new_password }),
}
