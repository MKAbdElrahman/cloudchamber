import { useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'

import { useMediaQuery } from '@astryxdesign/core/hooks'
import { LayoutPanel, VStack } from '@astryxdesign/core/Layout'
import { Banner } from '@astryxdesign/core/Banner'
import { Markdown } from '@astryxdesign/core/Markdown'
import { createMarkdownHeadingLinks } from '@astryxdesign/core/Markdown/plugins'
import { Outline, useOutlineFromMarkdown } from '@astryxdesign/core/Outline'
import { filesFromJobs, isProjectFileKind, loadDocumentBody, projectFileLabel, readingTitle } from '@/features/outputs'
import { PageHeader, PendingLines, SHELL_NARROW_QUERY, WorkspacePage } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { ProjectTrail } from './ProjectTrail'
import { TaskMark } from './TaskMark'
import { useProjectJobs } from './useProjectJobs'

type ProjectFilePageProps = {
  token: string
}

function nearestScrollParent(node: HTMLElement | null): HTMLElement | null {
  let current = node?.parentElement ?? null
  while (current) {
    const overflow = getComputedStyle(current).overflowY
    if (overflow === 'auto' || overflow === 'scroll') return current
    current = current.parentElement
  }
  return null
}

/** One document in a project. Contents sit in the side region; the task mark opens the pipeline. */
export function ProjectFilePage({ token }: ProjectFilePageProps) {
  const { projectId = '', jobId = '', kind: kindParam = '' } = useParams()
  const isNarrow = useMediaQuery(SHELL_NARROW_QUERY)
  const jobsQuery = useProjectJobs(token, projectId)
  const kind = isProjectFileKind(kindParam) ? kindParam : null
  const fileJob = jobsQuery.data?.find((job) => job.id === jobId)
  const file =
    kind == null
      ? null
      : (filesFromJobs(jobsQuery.data ?? []).find((item) => item.jobId === jobId && item.kind === kind) ?? null)
  const bodyQuery = useQuery({
    queryKey: ['document-body', file?.ref],
    queryFn: () => {
      if (!file) throw new Error('missing document')
      return loadDocumentBody(file)
    },
    enabled: file != null,
  })
  const markdown = bodyQuery.data?.markdown ?? ''
  const documentId = file?.id ?? 'document'
  const plugins = useMemo(
    () => [createMarkdownHeadingLinks({ headingIdPrefix: documentId })],
    [documentId],
  )
  const items = useOutlineFromMarkdown(markdown, { plugins })
  const [scrollRoot, setScrollRoot] = useState<HTMLElement | null>(null)
  const bindArticle = useCallback((node: HTMLElement | null) => {
    const next = nearestScrollParent(node)
    setScrollRoot((current) => (current === next ? current : next))
  }, [])
  const scrollContainerRef = useMemo(() => ({ current: scrollRoot }), [scrollRoot])
  const outline =
    items.length > 0 ? (
      <Outline
        items={items}
        density={isNarrow ? 'compact' : 'default'}
        label="Contents"
        scrollContainerRef={scrollContainerRef}
      />
    ) : null

  if (!kind) {
    return (
      <WorkspacePage header={<PageHeader title="Missing document" />}>
        <VStack gap={WORKSPACE_GAP.group}>
          <ProjectTrail projectId={projectId} current="Missing document" />
          <Banner status="error" title="Missing document" description="This address is not a document." />
        </VStack>
      </WorkspacePage>
    )
  }

  const title = file ? readingTitle(file.name) : jobsQuery.isPending ? 'Document' : 'Missing document'
  const ready = file != null && bodyQuery.data != null
  const sample =
    bodyQuery.data?.format === 'pdf'
      ? 'The PDF is not loaded from storage yet, so this is the document as text.'
      : 'The export is not loaded from storage yet, so this is a stand-in you can read.'

  return (
    <WorkspacePage
      start={
        !isNarrow && ready && outline ? (
          <LayoutPanel width="17.5rem" hasDivider>
            {outline}
          </LayoutPanel>
        ) : undefined
      }
      header={
        <VStack gap={WORKSPACE_GAP.group} width="100%">
          <ProjectTrail projectId={projectId} current={file?.name ?? 'Document'} />
          <PageHeader
            title={title}
            description={file ? projectFileLabel(file.kind) : undefined}
            end={
              file ? (
                <TaskMark projectId={projectId} jobId={file.jobId} status={fileJob?.status} />
              ) : null
            }
          />
        </VStack>
      }
    >
      <VStack ref={bindArticle} gap={WORKSPACE_GAP.group}>
        {jobsQuery.isPending || (file != null && bodyQuery.isPending) ? <PendingLines /> : null}
        {!jobsQuery.isPending && !file ? (
          <Banner status="error" title="Missing document" description="This project does not have that document." />
        ) : null}
        {ready && bodyQuery.data?.mocked ? <Banner status="info" title="Sample document" description={sample} /> : null}
        {isNarrow && ready ? outline : null}
        {ready ? (
          <Markdown id={documentId} plugins={plugins} headingLevelStart={2}>
            {markdown}
          </Markdown>
        ) : null}
      </VStack>
    </WorkspacePage>
  )
}
