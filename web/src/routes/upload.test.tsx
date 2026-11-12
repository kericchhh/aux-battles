import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { axeViolations, json, mockApi, renderRoute } from '@/test/render'
import { Upload } from './upload'

describe('Upload', () => {
  it('has no detectable accessibility violations', async () => {
    const { container } = renderRoute(<Upload />)
    expect(await axeViolations(container)).toEqual([])
  })

  it('links validation errors to their fields without calling the API', async () => {
    const fetch = mockApi({})
    renderRoute(<Upload />)

    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))

    expect(screen.getByLabelText('MP3 file')).toHaveAccessibleDescription(
      'Up to 30 MB, with at least 12 seconds after the clip start. Choose an MP3 file',
    )
    expect(screen.getByLabelText('Title')).toHaveAttribute('aria-invalid', 'true')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('uploads the song and shows its processing status', async () => {
    const fetch = mockApi({
      'POST /songs': () => json({ id: 'song-1', status: 'PROCESSING' }, 202),
      'GET /songs/song-1/status': () =>
        json({
          id: 'song-1',
          title: 'First Light',
          artist: 'Ana',
          status: 'READY',
          processingError: null,
          workerAvailable: true,
        }),
    })
    renderRoute(<Upload />)

    const file = new File(['ID3'], 'first.mp3', { type: 'audio/mpeg' })
    await userEvent.upload(screen.getByLabelText('MP3 file'), file)
    await userEvent.type(screen.getByLabelText('Title'), 'First Light')
    await userEvent.type(screen.getByLabelText('Artist'), 'Ana')
    await userEvent.type(screen.getByLabelText('Genre'), 'Pop')
    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))

    expect(await screen.findByText('Ready')).toHaveAttribute('role', 'status')
    const form = fetch.mock.calls.find(([, init]) => init?.method === 'POST')![1]!.body as FormData
    await waitFor(() => expect(form.get('title')).toBe('First Light'))
    expect(form.get('clipStartSeconds')).toBe('0')
    expect(form.get('song')).toHaveProperty('name', 'first.mp3')
  })
})
