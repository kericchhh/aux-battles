export function returnPath(state: unknown): string {
  const from = state && typeof state === 'object' && 'from' in state ? state.from : null
  if (
    typeof from !== 'string' ||
    !from.startsWith('/') ||
    from.startsWith('//') ||
    from.includes('\\') ||
    [...from].some((character) => character.charCodeAt(0) < 32)
  )
    return '/'
  const pathname = from.split(/[?#]/)[0]
  return pathname === '/sign-in' || pathname === '/register' ? '/' : from
}
