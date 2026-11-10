import { queryOptions } from '@tanstack/react-query'

import { api, isApiError } from './client'

export type User = {
  id: string
  username: string
  email: string
  role?: string
}

export const meQuery = queryOptions({
  queryKey: ['me'],
  queryFn: async () => {
    try {
      return await api<User>('/users/me')
    } catch (error) {
      if (isApiError(error, 401)) return null
      throw error
    }
  },
  staleTime: 5 * 60_000,
})

export const signIn = (body: { identifier: string; password: string }) =>
  api<User>('/users/login', { method: 'POST', body: JSON.stringify(body) })

export const register = (body: { username: string; email: string; password: string }) =>
  api<User>('/users', { method: 'POST', body: JSON.stringify(body) })

export const signOut = () => api<void>('/users/logout', { method: 'POST' })
