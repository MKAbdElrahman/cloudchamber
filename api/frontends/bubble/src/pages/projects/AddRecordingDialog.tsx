import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'

import { Banner } from '@astryxdesign/core/Banner'
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog'
import { VStack } from '@astryxdesign/core/Layout'
import { createJob, JobStatusLabel } from '@/features/conversion'
import { UploadPanel } from '@/features/upload'
import { ApiError } from '@/shared/api'
import { RECORDING_DIALOG_WIDTH } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

type AddRecordingDialogProps = {
  token: string
  projectId: string
  isOpen: boolean
  formKey: number
  onOpenChange: (isOpen: boolean) => void
  /** The recording is in the project. A message means the conversion did not start. */
  onFinished: (notice?: string) => void
  /** Set while bytes are transferring, cleared when the upload stops or the file is saved. */
  onUploadingFile: (fileName: string | null) => void
  /** Set when the dialog opened from a drop on the project. */
  initialFile?: File | null
}

function AddRecordingForm({
  token,
  projectId,
  onUploadingChange,
  onSavingChange,
  onUploadingFile,
  onFinished,
  initialFile,
}: {
  token: string
  projectId: string
  onUploadingChange: (uploading: boolean) => void
  onSavingChange: (saving: boolean) => void
  onUploadingFile: (fileName: string | null) => void
  onFinished: (notice?: string) => void
  initialFile?: File | null
}) {
  const queryClient = useQueryClient()
  const [startConversion, setStartConversion] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleUploaded(upload: { fileId: string; fileName: string }) {
    onSavingChange(true)
    setError(null)
    try {
      let notice: string | undefined
      if (startConversion) {
        try {
          const job = await createJob(token, projectId, upload.fileId, `ui-${crypto.randomUUID()}`)
          queryClient.setQueryData(['conversion', job.id], job)
        } catch (err) {
          notice = err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Could not start the conversion.'
        }
      }
      await queryClient.invalidateQueries({ queryKey: ['conversions', projectId] })
      await queryClient.invalidateQueries({ queryKey: ['project-files', projectId] })
      onUploadingFile(null)
      onFinished(notice)
    } catch (err) {
      onUploadingFile(null)
      setError(err instanceof Error ? err.message : 'Could not save the recording.')
    } finally {
      onSavingChange(false)
    }
  }

  return (
    <VStack gap={WORKSPACE_GAP.group}>
      {error ? <Banner status="error" title="Could not save the recording" description={error} /> : null}
      <UploadPanel
        token={token}
        projectId={projectId}
        startConversion={startConversion}
        onStartConversionChange={setStartConversion}
        onBusyChange={onUploadingChange}
        onUploadingFile={onUploadingFile}
        initialFile={initialFile}
        onUploaded={(upload) => {
          void handleUploaded(upload)
        }}
      />
    </VStack>
  )
}

/** Upload form over the project. The page underneath stays where it is. */
export function AddRecordingDialog({
  token,
  projectId,
  isOpen,
  formKey,
  onOpenChange,
  onFinished,
  onUploadingFile,
  initialFile,
}: AddRecordingDialogProps) {
  const [uploading, setUploading] = useState(false)
  const [saving, setSaving] = useState(false)
  const busy = uploading || saving

  function handleOpenChange(next: boolean) {
    if (!next && busy) return
    if (!next) {
      setUploading(false)
      setSaving(false)
      onUploadingFile(null)
    }
    onOpenChange(next)
  }

  return (
    <Dialog isOpen={isOpen} onOpenChange={handleOpenChange} purpose={busy ? 'required' : 'form'} width={RECORDING_DIALOG_WIDTH}>
      <DialogHeader
        title="Add recording"
        onOpenChange={handleOpenChange}
        endContent={uploading ? <JobStatusLabel status="uploading" /> : undefined}
      />
      <AddRecordingForm
        key={formKey}
        token={token}
        projectId={projectId}
        onUploadingChange={setUploading}
        onSavingChange={setSaving}
        onUploadingFile={onUploadingFile}
        onFinished={onFinished}
        initialFile={initialFile}
      />
    </Dialog>
  )
}
