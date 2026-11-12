import { CircleHelp } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import { StemMeter } from '@/components/game/stem-meter'

export function Rules() {
  return (
    <>
      <ol className="steps">
        <li>Host a battle of 1–7 rounds and share the invite code.</li>
        <li>Both players secretly pick one song per round.</li>
        <li>
          Each round you hear your opponent’s song one stem at a time. A wrong guess unlocks the
          next stem, up to four guesses.
        </li>
        <li>
          Earlier guesses score more. Consecutive correct rounds add 10 bonus points each, up to 20.
        </li>
        <li>With a timer on, running out of time on a stem counts as a wrong guess.</li>
      </ol>
      <div className="grid gap-2">
        <p className="text-dim">Points for a correct guess at each stem</p>
        <StemMeter className="stem-meter-inset" />
      </div>
    </>
  )
}

export function HowToPlay() {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="How to play" title="How to play">
          <CircleHelp aria-hidden />
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-lg">
        <DialogTitle>How to play</DialogTitle>
        <DialogDescription>
          Name your opponent’s song from as few stems as you can.
        </DialogDescription>
        <Rules />
      </DialogContent>
    </Dialog>
  )
}
