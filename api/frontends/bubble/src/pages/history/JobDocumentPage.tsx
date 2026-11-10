import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { Markdown } from '@astryxdesign/core/Markdown'
import { Outline, useOutlineFromMarkdown } from '@astryxdesign/core/Outline'
import { Heading, Text } from '@astryxdesign/core/Text'
import type { HeadingLevel } from '@astryxdesign/core/Heading'
import { isJobSuccess, usePipelineJob } from '@/features/pipeline'
import { ForeignProjectNotice, useJobOwner } from '@/features/projects'
import { PageHeader, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { ProjectTrail } from '../projects/ProjectTrail'
import { WORKSPACE_GAP } from '@/shared/spacing'
import { readingDocument } from './documentReading'
import { outputsForJob } from '../fixtures/mockOutputs'

const TimeByHeading = createContext<ReadonlyMap<string, string>>(new Map())

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

function nodeText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(nodeText).join('')
  if (node && typeof node === 'object' && 'props' in node) {
    const element = node as { props?: { children?: ReactNode } }
    return nodeText(element.props?.children)
  }
  return ''
}

function DocumentHeading({
  level,
  children,
  id,
}: {
  level: HeadingLevel
  children: ReactNode
  id?: string
}) {
  const time = useContext(TimeByHeading).get(nodeText(children).trim())
  return (
    <VStack id={id} gap={0.5}>
      {time ? <Text type="supporting">{time}</Text> : null}
      <Heading level={level}>{children}</Heading>
    </VStack>
  )
}

/** The file a finished job produced. The job stays behind View job. */
export function JobDocumentPage({ token }: JobDocumentPageProps) {
  const navigate = useNavigate()
  const { jobId: jobIdParam, projectId = '' } = useParams()
  const jobId = jobIdParam?.trim() ?? ''
  const owner = useJobOwner(jobId)
  const inProject = owner.data === projectId
  const jobQuery = usePipelineJob(token, inProject ? jobId || null : null)
  const job = jobQuery.data
  const file = useMemo(() => {
    if (!job || !isJobSuccess(job.status)) return undefined
    return outputsForJob(job.id, job.status).find((item) => item.markdown)
  }, [job])
  const document = useMemo(() => readingDocument(file?.markdown ?? ''), [file?.markdown])
  const outline = useOutlineFromMarkdown(document.body)
  const title =
    owner.isSuccess && !inProject ? 'Another project' : (document.title ?? file?.name ?? 'Document')

  return (
    <WorkspacePage
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current={title} />
          <PageHeader
            title={title}
            description={document.dek}
            end={
              inProject && job ? (
                <HStack gap={WORKSPACE_GAP.local} vAlign="center">
                  {file?.markdown ? (
                    <Button
                      label="Markdown"
                      variant="secondary"
                      onClick={() => downloadMarkdown(file.name, file.markdown ?? '')}
                    />
                  ) : null}
                  <Button
                    label="View job"
                    variant="secondary"
                    onClick={() => navigate(WORKSPACE_PATHS.historyJob(projectId, job.id))}
                  />
                </HStack>
              ) : undefined
            }
          />
        </VStack>
      }
    >
      {owner.data && owner.data !== projectId ? (
        <ForeignProjectNotice ownerId={owner.data} jobId={jobId} />
      ) : null}
      {inProject && jobQuery.isError ? (
        <Banner
          status="error"
          title="Could not load job"
          description={(jobQuery.error as Error).message}
        />
      ) : null}
      {owner.isPending || (inProject && jobQuery.isPending && !job) ? (
        <Text type="supporting">Loading document…</Text>
      ) : null}
      {inProject && job && !file ? (
        <Text color="secondary">This run does not have a document yet.</Text>
      ) : null}
      {inProject && file ? (
        <HStack gap={WORKSPACE_GAP.section} vAlign="start" width="100%">
          {outline.length > 0 ? (
            <VStack gap={WORKSPACE_GAP.local} width={220}>
              <Text type="label">In this document</Text>
              <Outline items={outline} label="In this document" />
            </VStack>
          ) : null}
          <TimeByHeading.Provider value={document.timeByHeading}>
            <Markdown components={{ heading: DocumentHeading }}>{document.body}</Markdown>
          </TimeByHeading.Provider>
        </HStack>
      ) : null}
    </WorkspacePage>
  )
}
