import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Catalog } from './catalog'
import { LineupPicker } from './lineup-picker'
import { battle, songs } from '@/test/fixtures'
import { json, mockFetch, renderRoute } from '@/test/render'

describe('catalog and lineup', () => {
  it('keeps keyboard focus on both move buttons after reorder, including the last position', async () => {
    mockFetch(() => json({ songs, nextCursor: null }))
    renderRoute(<LineupPicker battle={battle({ status: 'SELECTING', myLineupLocked: false })} />)
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Add First Light' }))
    await user.click(screen.getByRole('button', { name: 'Add Second Wind' }))
    expect(screen.getByRole('button', { name: 'Added First Light' })).toHaveAttribute(
      'aria-disabled',
      'true',
    )
    const down = screen.getByRole('button', { name: 'Move First Light down' })
    down.focus()
    await user.keyboard('{Enter}')
    expect(down).toHaveFocus()
    expect(down).toHaveAttribute('aria-disabled', 'true')
    const up = screen.getByRole('button', { name: 'Move First Light up' })
    up.focus()
    await user.keyboard('{Enter}')
    expect(up).toHaveFocus()
    expect(up).toHaveAttribute('aria-disabled', 'true')
  })

  it('waits for debounced search results before claiming no match', async () => {
    let resolveSearch: ((response: Response) => void) | undefined
    mockFetch((url) =>
      url.includes('q=wind')
        ? new Promise<Response>((resolve) => {
            resolveSearch = resolve
          })
        : json({ songs: [], nextCursor: null }),
    )
    renderRoute(<Catalog rounds={2} picked={[]} full={false} onAdd={() => {}} />)
    await screen.findByText(/catalog is empty/)
    await userEvent.type(screen.getByLabelText('Search songs'), 'wind')
    expect(screen.queryByText('No songs match “wind”.')).not.toBeInTheDocument()
    await screen.findByText('Searching songs…')
    await waitFor(() => expect(resolveSearch).toBeDefined())
    await act(async () => {
      resolveSearch!(json({ songs: [], nextCursor: null }))
    })
    // Ensure the debounced request has started before resolving it.
    await waitFor(() => expect(screen.queryByText('No songs match “wind”.')).toBeInTheDocument())
  })

  it('appends the next page and retains songs if loading more fails', async () => {
    let fail = false
    mockFetch((url) =>
      url.includes('cursor=')
        ? fail
          ? json({ message: 'unavailable' }, 500)
          : json({ songs: [songs[1]], nextCursor: 'last' })
        : json({ songs: [songs[0]], nextCursor: 'next' }),
    )
    renderRoute(<Catalog rounds={2} picked={[]} full={false} onAdd={() => {}} />)
    await userEvent.click(await screen.findByRole('button', { name: 'Load more' }))
    expect(await screen.findByText('Second Wind')).toBeInTheDocument()
    fail = true
    await userEvent.click(screen.getByRole('button', { name: 'Load more' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Load more' })).toBeEnabled())
    expect(screen.getByText('First Light')).toBeInTheDocument()
    expect(screen.getByText('Second Wind')).toBeInTheDocument()
  })
})
