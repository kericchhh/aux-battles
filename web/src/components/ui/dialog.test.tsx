import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HowToPlay } from '@/components/game/how-to-play'
import { axeViolations, renderRoute } from '@/test/render'

it('traps focus in the help dialog and restores it on Escape', async () => {
  const { container } = renderRoute(<HowToPlay />)
  const trigger = screen.getByRole('button', { name: 'How to play' })
  const user = userEvent.setup()
  await user.click(trigger)
  const dialog = await screen.findByRole('dialog', { name: 'How to play' })
  const close = within(dialog).getByRole('button', { name: 'Close' })
  expect(close).toHaveFocus()
  await user.keyboard('{Tab}')
  expect(close).toHaveFocus()
  expect(await axeViolations(dialog)).toEqual([])
  await user.keyboard('{Escape}')
  expect(trigger).toHaveFocus()
  expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument()
})
