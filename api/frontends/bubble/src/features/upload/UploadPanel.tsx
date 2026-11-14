import { useEffect, useMemo, useRef, useState } from 'react'
import * as tus from 'tus-js-client'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { FileInput } from '@astryxdesign/core/FileInput'
import { ProgressBar } from '@astryxdesign/core/ProgressBar'
import { Banner } from '@astryxdesign/core/Banner'
import { CheckboxInput } from '@astryxdesign/core/CheckboxInput'
import { env } from '@/shared/config'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { fileIdFromUploadUrl } from './sourceRef'

type Status = 'idle' | 'uploading' | 'success' | 'warning' | 'error'

export function UploadPanel({
  token,
  projectId,
  onUploaded,
  startConversion,
  onStartConversionChange,
  onBusyChange,
  onUploadingFile,
  initialFile,
}: {
  token: string
  projectId: string
  onUploaded: (upload: { fileId: string; fileName: string }) => void
  startConversion: boolean
  onStartConversionChange: (checked: boolean) => void
  onBusyChange?: (busy: boolean) => void
  /** The file currently transferring, so the project can show Uploading first. */
  onUploadingFile?: (fileName: string | null) => void
  /** A file dropped on the project, already chosen when the dialog opens. */
  initialFile?: File | null
}) {
  const [file, setFile] = useState<File | null>(initialFile ?? null)
  const [status, setStatus] = useState<Status>('idle')
  const [progress, setProgress] = useState(0)
  const [message, setMessage] = useState<string | null>(null)
  const uploadRef = useRef<tus.Upload | null>(null)
  const canStart = useMemo(() => !!file && status !== 'uploading', [file, status])

  useEffect(() => {
    onBusyChange?.(status === 'uploading')
  }, [onBusyChange, status])

  function resetUploadState() {
    setStatus('idle')
    setProgress(0)
    setMessage(null)
  }

  function start() {
    if (!file) return
    setStatus('uploading')
    setProgress(0)
    setMessage(null)
    onUploadingFile?.(file.name)

    const upload = new tus.Upload(file, {
      endpoint: env.tusdEndpoint,
      retryDelays: [0, 1000, 3000, 5000, 10000],
      metadata: {
        filename: file.name,
        filetype: file.type || 'application/octet-stream',
        projectId,
      },
      headers: { Authorization: `Bearer ${token}` },
      onError(err) {
        setStatus('error')
        setMessage(err.message || 'Upload failed')
        onUploadingFile?.(null)
      },
      onProgress(bytesUploaded, bytesTotal) {
        if (bytesTotal > 0) setProgress((bytesUploaded / bytesTotal) * 100)
      },
      onSuccess() {
        const fileId = upload.url ? fileIdFromUploadUrl(upload.url) : null
        setProgress(100)
        if (fileId) {
          setStatus('success')
          setMessage(null)
          onUploaded({ fileId, fileName: file.name })
        } else {
          setStatus('warning')
          setMessage('')
        }
      },
    })

    uploadRef.current = upload
    upload.findPreviousUploads().then((previous) => {
      if (previous.length) upload.resumeFromPreviousUpload(previous[0])
      upload.start()
    })
  }

  return (
    <VStack gap={WORKSPACE_GAP.section}>
      <VStack gap={WORKSPACE_GAP.group}>
        <FileInput
          label="Recording"
          accept="video/*,audio/*"
          mode="dropzone"
          value={file}
          onChange={(files) => {
            const next = files && !Array.isArray(files) ? files : Array.isArray(files) ? files[0] ?? null : null
            setFile(next)
            resetUploadState()
          }}
          isLoading={status === 'uploading'}
          statusVariant="detached"
        />

        {status === 'uploading' || progress > 0 ? (
          <ProgressBar label="Upload progress" value={progress} hasValueLabel variant="accent" />
        ) : null}

        <CheckboxInput
          label="Create the document after upload"
          value={startConversion}
          onChange={onStartConversionChange}
          isDisabled={status === 'uploading'}
        />
      </VStack>

      {message || status === 'warning' ? (
        <Banner
          status={status === 'warning' ? 'warning' : 'error'}
          title={status === 'warning' ? 'Recording was not saved' : 'Upload failed'}
          description={message || undefined}
        />
      ) : null}

      <Button
        type="button"
        label={status === 'uploading' ? 'Uploading…' : 'Start upload'}
        variant="primary"
        icon={<Icon icon={ArrowUpTrayIcon} />}
        onClick={start}
        isDisabled={!canStart}
        isLoading={status === 'uploading'}
      />
    </VStack>
  )
}
