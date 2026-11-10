import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { VStack } from '@astryxdesign/core/Layout'
import { Heading, Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'
import { TextInput } from '@astryxdesign/core/TextInput'
import { Banner } from '@astryxdesign/core/Banner'
import { FormLayout } from '@astryxdesign/core/FormLayout'
import { ApiError } from '@/shared/api/client'
import { createJob } from './api'
import type { PipelineJob } from './types'

export function CreateJobForm({
  token,
  sourceRef: initialSourceRef,
  embedded,
  onJobCreated,
}: {
  token: string
  sourceRef: string | null
  embedded?: boolean
  onJobCreated?: (job: PipelineJob) => void
}) {
  const qc = useQueryClient()
  const [sourceRef, setSourceRef] = useState(initialSourceRef ?? '')

  useEffect(() => {
    if (initialSourceRef) setSourceRef(initialSourceRef)
  }, [initialSourceRef])

  const create = useMutation({
    mutationFn: (sr: string) => createJob(token, sr, `ui-${crypto.randomUUID()}`),
    onSuccess: (job: PipelineJob) => {
      qc.setQueryData(['pipeline-job', job.id], job)
      qc.invalidateQueries({ queryKey: ['pipeline-jobs'] })
      onJobCreated?.(job)
    },
  })

  const body = (
    <VStack gap={3}>
      {embedded ? null : (
        <VStack gap={1}>
          <Heading level={3}>Create job</Heading>
          <Text type="supporting" color="secondary" size="sm">
            Uses your finalized source reference from upload.
          </Text>
        </VStack>
      )}
      {embedded && !initialSourceRef ? (
        <Banner
          status="info"
          title="Upload a recording first"
          description="Complete the Upload stage so a source reference is available here."
        />
      ) : null}
      <FormLayout>
        <TextInput label="Source reference" value={sourceRef} onChange={setSourceRef} />
      </FormLayout>
      {create.isError ? (
        <Banner
          status="error"
          title="Could not create job"
          description={
            create.error instanceof ApiError ? create.error.message : (create.error as Error).message
          }
        />
      ) : null}
      <Button
        label={create.isPending ? 'Creating…' : 'Create job'}
        variant="primary"
        onClick={() => create.mutate(sourceRef.trim())}
        isDisabled={!sourceRef.trim() || create.isPending}
        isLoading={create.isPending}
      />
    </VStack>
  )

  if (embedded) return body

  return body
}
