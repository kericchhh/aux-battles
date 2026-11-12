import { returnPath } from './return-path'

it.each([
  [{ from: '/upload' }, '/upload'],
  [{ from: '/battles/one' }, '/battles/one'],
  [null, '/'],
  [{ from: 'https://example.test' }, '/'],
  [{ from: '//example.test' }, '/'],
  [{ from: '/\\example.test' }, '/'],
  [{ from: '/sign-in' }, '/'],
  [{ from: '/register?next=/upload' }, '/'],
])('validates return destinations: %j', (state, destination) => {
  expect(returnPath(state)).toBe(destination)
})
