import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'

import { meQuery } from './auth'
import { isApiError } from './client'

export function createQueryClient() {
  const expireSession = (error: Error) => {
    if (isApiError(error, 401)) client.setQueryData(meQuery.queryKey, null)
  }
  const client = new QueryClient({
    queryCache: new QueryCache({
      onError: expireSession,
    }),
    mutationCache: new MutationCache({
      onError: (error, _variables, _context, mutation) => {
        if (mutation.options.meta?.requiresAuth !== false) expireSession(error)
      },
    }),
    defaultOptions: {
      queries: {
        retry: (count, error) => !isApiError(error) && count < 2,
        refetchOnWindowFocus: true,
      },
    },
  })
  return client
}
