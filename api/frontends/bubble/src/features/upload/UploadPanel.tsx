import { useMemo, useRef, useState } from 'react'
import * as tus from 'tus-js-client'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { Card } from '@astryxdesign/core/Card'
import { TextInput } from '@astryxdesign/core/TextInput'
import { FileInput } from '@astryxdesign/core/FileInput'
import { ProgressBar } from '@astryxdesign/core/ProgressBar'
import { Banner } from '@astryxdesign/core/Banner'
import { Switch } from '@astryxdesign/core/Switch'
import { env } from '@/shared/config'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/spacing'
import { sourceRefFromUploadUrl } from './sourceRef'

type Status = 'idle' | 'uploading' | 'success' | 'error'

export function UploadPanel({
  token,
  onSourceRef,
  embedded,
}: {
  token: string
  onSourceRef: (sourceRef: string) => void
  embedded?: boolean
}) {
  const [file, setFile] = useState<File | null>(null)
  const [endpoint, setEndpoint] = useState(env.tusdEndpoint)
  const [showEndpoint, setShowEndpoint] = useState(false)
  const [status, setStatus] = useState<Status>('idle')
  const [progress, setProgress] = useState(0)
  const [message, setMessage] = useState<string | null>(null)
  const [sourceRef, setSourceRef] = useState<string | null>(null)
  const uploadRef = useRef<tus.Upload | null>(null)
  const canStart = useMemo(() => !!file && status !== 'uploading', [file, status])

  function resetUploadState() {
    setStatus('idle')
    setProgress(0)
    setMessage(null)
    setSourceRef(null)
  }

  function start() {
    if (!file) return
    setStatus('uploading')
    setProgress(0)
    setMessage(null)
    setSourceRef(null)

    const upload = new tus.Upload(file, {
      endpoint,
      retryDelays: [0, 1000, 3000, 5000, 10000],
      metadata: { filename: file.name, filetype: file.type || 'application/octet-stream' },
      headers: { Authorization: `Bearer ${token}` },
      onError(err) {
        setStatus('error')
        setMessage(err.message || 'Upload failed')
      },
      onProgress(bytesUploaded, bytesTotal) {
        if (bytesTotal > 0) setProgress((bytesUploaded / bytesTotal) * 100)
      },
      onSuccess() {
        const sr = upload.url ? sourceRefFromUploadUrl(upload.url) : null
        setStatus('success')
        setProgress(100)
        if (sr) {
          setSourceRef(sr)
          setMessage('Upload complete and finalized. Source reference is ready.')
          onSourceRef(sr)
        } else {
          setMessage(`Upload complete at ${upload.url ?? '(unknown url)'}, but the sourceRef could not be derived.`)
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
        <Switch label="Custom tus endpoint" value={showEndpoint} onChange={setShowEndpoint} />
        {showEndpoint ? <TextInput label="tusd endpoint" value={endpoint} onChange={setEndpoint} /> : null}

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
      </VStack>

      {message ? (
        <Banner
          status={status === 'error' ? 'error' : 'success'}
          title={status === 'error' ? 'Upload failed' : 'Ready'}
          description={
            sourceRef ? (
              <>
                {message} <Text type="code">{sourceRef}</Text>
              </>
            ) : (
              message
            )
          }
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
