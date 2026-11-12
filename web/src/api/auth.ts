import { queryOptions, type QueryClient } from '@tanstack/react-query'

import { api, isApiError } from './client'

export type User = {
  id: string
  username: string
  email: string
  role?: string
}

export const meQuery = queryOptions({
  queryKey: ['me'],
  queryFn: async ({ signal }) => {
    try {
      return await api<User>('/users/me', { signal })
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

export async function acceptSession(client: QueryClient) {
  await client.cancelQueries({ queryKey: meQuery.queryKey, exact: true })
  client.removeQueries({ predicate: (query) => query.queryKey[0] !== 'me' })
  const user = await client.fetchQuery({ ...meQuery, staleTime: 0 })
  if (!user) throw new Error('Could not confirm the session. Please sign in again.')
  return user
}
