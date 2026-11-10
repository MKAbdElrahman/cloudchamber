import { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { ChevronRightIcon } from '@heroicons/react/24/outline'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { BreadcrumbItem, Breadcrumbs } from '@astryxdesign/core/Breadcrumbs'
import { Button } from '@astryxdesign/core/Button'
import { Icon } from '@astryxdesign/core/Icon'
import { Link } from '@astryxdesign/core/Link'
import { Heading, Text } from '@astryxdesign/core/Text'
import { isJobSuccess, usePipelineJob } from '@/features/pipeline'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { fileNameFromSourceRef } from '@/shared/media'

import { parseDocument, type DocumentBlock } from './documentMarkdown'
import { outputsForJob } from '../fixtures/mockOutputs'

type JobDocumentPageProps = {
  token: string
}

function downloadMarkdown(name: string, markdown: string) {
  const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = name
  anchor.click()
  URL.revokeObjectURL(url)
}

function DocumentBody({ blocks }: { blocks: DocumentBlock[] }) {
  return (
    <VStack gap={WORKSPACE_GAP.group}>
      {blocks.map((block, index) => {
        if (block.kind === 'heading') {
          return (
            <VStack key={block.id} id={block.id} gap={0.5}>
              {block.time ? <Text type="supporting">{block.time}</Text> : null}
              <Heading level={block.level === 1 ? 2 : block.level}>{block.text}</Heading>
            </VStack>
          )
        }
        if (block.kind === 'quote') {
          return (
            <Text key={`quote-${index}`} color="secondary">
              {block.text}
            </Text>
          )
        }
        return <Text key={`p-${index}`}>{block.text}</Text>
      })}
    </VStack>
  )
}

/** The one document a finished job produces. */
export function JobDocumentPage({ token }: JobDocumentPageProps) {
  const { jobId: jobIdParam } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const jobQuery = usePipelineJob(token, jobId || null)
  const job = jobQuery.data
  const file = useMemo(() => {
    if (!job || !isJobSuccess(job.status)) return undefined
    return outputsForJob(job.id, job.status).find((item) => item.markdown)
  }, [job])
  const blocks = useMemo(() => parseDocument(file?.markdown ?? ''), [file?.markdown])
  const titleBlock = blocks.find((block) => block.kind === 'heading' && block.level === 1)
  const title = titleBlock?.kind === 'heading' ? titleBlock.text : file?.name ?? 'Document'
  const dek = blocks.find((block) => block.kind === 'paragraph')
  const body = blocks.filter((block) => block !== titleBlock && block !== dek)
  const outline = body.filter((block) => block.kind === 'heading' && block.level === 2)
  const jobLabel = job ? fileNameFromSourceRef(job.sourceRef) : jobId

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <Breadcrumbs
            variant="supporting"
            separator={<Icon icon={ChevronRightIcon} size="sm" color="secondary" />}
            aria-label="Document navigation"
          >
            <BreadcrumbItem href={WORKSPACE_PATHS.history}>History</BreadcrumbItem>
            <BreadcrumbItem href={WORKSPACE_PATHS.historyJob(jobId)}>{jobLabel}</BreadcrumbItem>
            <BreadcrumbItem isCurrent>Document</BreadcrumbItem>
          </Breadcrumbs>
          <PageHeader
            title={title}
            description={dek?.kind === 'paragraph' ? dek.text : undefined}
            end={
              file?.markdown ? (
                <Button
                  label="Markdown"
                  variant="secondary"
                  onClick={() => downloadMarkdown(file.name, file.markdown ?? '')}
                />
              ) : undefined
            }
          />
        </VStack>
      }
    >
      {jobQuery.isError ? (
        <Banner
          status="error"
          title="Could not load job"
          description={(jobQuery.error as Error).message}
        />
      ) : null}
      {jobQuery.isPending && !job ? <Text type="supporting">Loading document…</Text> : null}
      {job && !file ? (
        <Text color="secondary">This run does not have a document yet.</Text>
      ) : null}
      {file ? (
        <HStack gap={WORKSPACE_GAP.section} vAlign="start" wrap="wrap">
          {outline.length > 0 ? (
            <VStack gap={1} aria-label="In this document">
              <Text type="label">In this document</Text>
              {outline.map((block) =>
                block.kind === 'heading' ? (
                  <Link key={block.id} href={`#${block.id}`}>
                    {block.text}
                  </Link>
                ) : null,
              )}
            </VStack>
          ) : null}
          <VStack gap={WORKSPACE_GAP.group}>
            <DocumentBody blocks={body} />
          </VStack>
        </HStack>
      ) : null}
    </WorkspacePage>
  )
}
