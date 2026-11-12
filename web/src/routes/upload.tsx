import { useState } from 'react'

import { PageHeader } from '@/components/page-header'
import { Card } from '@/components/ui/card'
import { UploadForm } from '@/features/upload/upload-form'
import { UploadStatus } from '@/features/upload/upload-status'

export function Upload() {
  const [uploads, setUploads] = useState<string[]>([])

  return (
    <div className="grid gap-8">
      <PageHeader
        title="Upload a song"
        description="Only upload tracks you have permission to store, process and play in battles."
      />
      <div className="grid items-start gap-5 lg:grid-cols-12">
        <UploadForm
          className="lg:col-span-7"
          onUploaded={(id) => setUploads((current) => [id, ...current])}
        />
        <Card title="Processing" className="lg:col-span-5">
          {uploads.length === 0 ? (
            <p className="text-dim">
              Songs you upload here show their progress until they are ready.
            </p>
          ) : (
            <ul className="grid gap-1">
              {uploads.map((id) => (
                <UploadStatus key={id} id={id} />
              ))}
            </ul>
          )}
        </Card>
      </div>
    </div>
  )
}
