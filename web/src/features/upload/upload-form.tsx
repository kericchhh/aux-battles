import { useMutation } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { uploadSong } from '@/api/songs'
import { Field } from '@/components/field'
import { FormError } from '@/components/form-error'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

const textField = (max: number) =>
  z.string().trim().min(1, 'Required').max(max, `At most ${max} characters`)
const schema = z.object({
  song: z
    .custom<FileList>()
    .refine((files) => Boolean(files?.[0]?.size), 'Choose an MP3 file')
    .refine(
      (files) => !files?.[0] || files[0].size <= 30 * 1024 * 1024,
      'The file must be 30 MB or smaller',
    ),
  title: textField(255),
  artist: textField(255),
  genre: textField(50),
  clipStartSeconds: z
    .string()
    .trim()
    .refine(
      (value) => value !== '' && Number.isFinite(Number(value)) && Number(value) >= 0,
      'Enter a start time of 0 or more seconds',
    ),
})

export function UploadForm({
  className,
  onUploaded,
}: {
  className?: string
  onUploaded: (id: string) => void
}) {
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { clipStartSeconds: '0' } })
  const errors = form.formState.errors
  const upload = useMutation({ mutationFn: uploadSong })
  function submit(values: z.output<typeof schema>) {
    if (upload.isPending) return
    const body = new FormData()
    body.set('song', values.song[0]!)
    for (const name of ['title', 'artist', 'genre', 'clipStartSeconds'] as const)
      body.set(name, values[name])
    upload.mutate(body, {
      onSuccess: (song) => {
        form.reset()
        onUploaded(song.id)
      },
    })
  }

  return (
    <Card title="Song details" className={className}>
      <form noValidate onSubmit={form.handleSubmit(submit)} className="grid gap-5">
        <Field
          label="MP3 file"
          {...form.register('song')}
          type="file"
          accept="audio/mpeg,.mp3"
          hint="Up to 30 MB, with at least 12 seconds after the clip start."
          error={errors.song?.message}
        />
        <div className="grid gap-5 sm:grid-cols-2">
          <Field
            label="Title"
            {...form.register('title')}
            maxLength={255}
            autoComplete="off"
            error={errors.title?.message}
          />
          <Field
            label="Artist"
            {...form.register('artist')}
            maxLength={255}
            autoComplete="off"
            error={errors.artist?.message}
          />
          <Field
            label="Genre"
            {...form.register('genre')}
            maxLength={50}
            autoComplete="off"
            error={errors.genre?.message}
          />
          <Field
            label="Clip start (seconds)"
            {...form.register('clipStartSeconds')}
            type="number"
            inputMode="decimal"
            min={0}
            step="any"
            className="font-mono"
            error={errors.clipStartSeconds?.message}
          />
        </div>
        <FormError message={upload.error?.message} />
        <Button type="submit" disabled={upload.isPending} className="w-fit">
          {upload.isPending ? 'Uploading…' : 'Upload'}
        </Button>
      </form>
    </Card>
  )
}
