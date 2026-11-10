import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { axeViolations, json, mockFetch, renderRoute } from '@/test/render'
import { Home } from './home'

describe('Home', () => {
  it('has no detectable accessibility violations', async () => {
    const { container } = renderRoute(<Home />)
    expect(await axeViolations(container)).toEqual([])
  })

  it('creates a battle with the chosen round count', async () => {
    const fetch = mockFetch(() => json({ id: 'b1', inviteCode: 'ABC123' }, 201))
    const { router } = renderRoute(<Home />)

    await userEvent.click(screen.getByRole('radio', { name: '3' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create battle' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/battles/b1'))
    expect(JSON.parse(String(fetch.mock.calls[0]![1]?.body))).toEqual({ rounds: 3 })
  })

  it('shows the most points the chosen round count allows', async () => {
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('radio', { name: '3' }))

    expect(screen.getByText(/Up to/)).toHaveTextContent('Up to 330 points')
  })

  it('rejects malformed invite codes without calling the API', async () => {
    const fetch = mockFetch(() => json({}))
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('tab', { name: 'Join' }))
    await userEvent.type(screen.getByLabelText('Invite code'), 'ab1')
    await userEvent.click(screen.getByRole('button', { name: 'Join battle' }))

    expect(screen.getByLabelText('Invite code')).toHaveAccessibleDescription(
      'Ask the host for their 6-character code. Codes are 6 letters or numbers',
    )
    expect(fetch).not.toHaveBeenCalled()
  })

  it('explains an unknown invite code', async () => {
    mockFetch(() => json({ message: 'battle not found' }, 404))
    renderRoute(<Home />)

    await userEvent.click(screen.getByRole('tab', { name: 'Join' }))
    await userEvent.type(screen.getByLabelText('Invite code'), 'zzz999')
    await userEvent.click(screen.getByRole('button', { name: 'Join battle' }))

    expect(await screen.findByText('No battle uses that code.')).toBeInTheDocument()
  })
})
