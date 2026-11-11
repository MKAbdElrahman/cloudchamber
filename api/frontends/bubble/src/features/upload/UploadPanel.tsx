import { useEffect, useMemo, useRef, useState } from 'react'
import * as tus from 'tus-js-client'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { Card } from '@astryxdesign/core/Card'
import { FileInput } from '@astryxdesign/core/FileInput'
import { ProgressBar } from '@astryxdesign/core/ProgressBar'
import { Banner } from '@astryxdesign/core/Banner'
import { CheckboxInput } from '@astryxdesign/core/CheckboxInput'
import { readTusdEndpoint } from '@/shared/config'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { fileIdFromUploadUrl } from './sourceRef'

type Status = 'idle' | 'uploading' | 'success' | 'error'

export function UploadPanel({
  token,
  onUploaded,
  startConversion,
  onStartConversionChange,
  onBusyChange,
  onUploadingFile,
  embedded,
}: {
  token: string
  onUploaded: (upload: { fileId: string; fileName: string }) => void
  startConversion: boolean
  onStartConversionChange: (checked: boolean) => void
  onBusyChange?: (busy: boolean) => void
  /** The file currently transferring, so the project can show Uploading first. */
  onUploadingFile?: (fileName: string | null) => void
  embedded?: boolean
}) {
  const [file, setFile] = useState<File | null>(null)
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
      endpoint: readTusdEndpoint(),
      retryDelays: [0, 1000, 3000, 5000, 10000],
      metadata: { filename: file.name, filetype: file.type || 'application/octet-stream' },
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
        setStatus('success')
        setProgress(100)
        if (fileId) {
          setMessage(null)
          onUploaded({ fileId, fileName: file.name })
        } else {
          setMessage(`Upload complete at ${upload.url ?? '(unknown url)'}, but the recording could not be saved.`)
        }
      },
    })

    uploadRef.current = upload
    upload.findPreviousUploads().then((previous) => {
      if (previous.length) upload.resumeFromPreviousUpload(previous[0])
      upload.start()
    })
  }

  const content = (
    <VStack gap={WORKSPACE_GAP.section}>
      {embedded ? null : (
        <VStack gap={WORKSPACE_GAP.local}>
          <Heading level={2}>Upload recording</Heading>
          <Text type="supporting">
            Drop video or audio — resumable tus transfer with server-side verification.
          </Text>
        </VStack>
      )}

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
          label="Create the document when the upload finishes"
          description="Leave this off to watch the recording first, then start the conversion yourself."
          value={startConversion}
          onChange={onStartConversionChange}
          isDisabled={status === 'uploading'}
        />
      </VStack>

      {message ? (
        <Banner
          status={status === 'error' ? 'error' : 'success'}
          title={status === 'error' ? 'Upload failed' : 'Ready'}
          description={message}
        />
      ) : null}

      <Button
        type="button"
        label={status === 'uploading' ? 'Uploading…' : 'Start upload'}
        variant="primary"
        size="lg"
        icon={<Icon icon={ArrowUpTrayIcon} />}
        onClick={start}
        isDisabled={!canStart}
        isLoading={status === 'uploading'}
      />
    </VStack>
  )

  if (embedded) return content

  return (
    <Card {...WORKSPACE_CARD}>
      {content}
    </Card>
  )
}
