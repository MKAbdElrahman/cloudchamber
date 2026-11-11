import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderIcon } from '@heroicons/react/24/outline'

import { Grid } from '@astryxdesign/core/Grid'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { AlertDialog } from '@astryxdesign/core/AlertDialog'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { ClickableCard } from '@astryxdesign/core/ClickableCard'
import { Divider } from '@astryxdesign/core/Divider'
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { Icon } from '@astryxdesign/core/Icon'
import { List, ListItem } from '@astryxdesign/core/List'
import { MoreMenu } from '@astryxdesign/core/MoreMenu'
import { Heading, Text } from '@astryxdesign/core/Text'
import { TextInput } from '@astryxdesign/core/TextInput'
import {
  createProject,
  deleteProject,
  listProjects,
  projectErrorMessage,
  updateProject,
  useProjects,
  type Project,
} from '@/features/projects'
import { listJobs, normalizeJobs, type PipelineJob } from '@/features/pipeline'
import { filesFromJob } from '@/features/outputs'
import { PAGE_CONTENT_WIDTH, PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_CARD, WORKSPACE_CARD_FLUSH, WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR } from '@/shared/theme'

import { LibraryViewControl } from './LibraryViewControl'
import { useLibraryView } from './useLibraryView'

type ProjectsHomePageProps = {
  token: string
}

function countLabel(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`
}

type ProjectTileProps = {
  project: Project
  files: number
  countsReady: boolean
  onRename: (project: Project) => void
  onDelete: (project: Project) => void
}

function projectSummary(files: number, countsReady: boolean): string | null {
  if (!countsReady) return null
  if (files === 0) return 'No documents yet.'
  return `${countLabel(files, 'document', 'documents')}.`
}

type ProjectScope = {
  jobsByProject: Record<string, PipelineJob[]>
}

function projectFileCount(
  projectId: string,
  scope: ProjectScope | undefined,
  countsPending: boolean,
): number {
  if (countsPending || !scope) return 0
  return (scope.jobsByProject[projectId] ?? []).reduce((count, job) => count + filesFromJob(job).length, 0)
}

/** One project card. The folder mark names it; the card opens it. */
function ProjectTile({ project, files, countsReady, onRename, onDelete }: ProjectTileProps) {
  const summary = projectSummary(files, countsReady)

  return (
    <ClickableCard
      label={`Open ${project.name}`}
      href={WORKSPACE_PATHS.project(project.id)}
      {...WORKSPACE_CARD_FLUSH}
      height="100%"
    >
      <VStack gap={0} height="100%">
        <HStack gap={WORKSPACE_GAP.group} padding={WORKSPACE_CARD.padding} justify="between" vAlign="center">
          <HStack gap={WORKSPACE_GAP.group} vAlign="center">
            <Icon icon={FolderIcon} size="lg" color={MUTED_ICON_COLOR} />
            <Heading level={3} maxLines={1}>
              {project.name}
            </Heading>
          </HStack>
          <MoreMenu
            label={`Manage ${project.name}`}
            size="sm"
            alignment="end"
            items={[
              { label: 'Rename', onClick: () => onRename(project) },
              { type: 'divider' },
              { label: 'Delete', variant: 'destructive', onClick: () => onDelete(project) },
            ]}
          />
        </HStack>
        <Divider />
        <VStack padding={WORKSPACE_CARD.padding}>
          <Text type="supporting">
            {summary ?? ' '}
          </Text>
        </VStack>
      </VStack>
    </ClickableCard>
  )
}

/** Home — project tiles you open, rename, or delete. */
export function ProjectsHomePage({ token }: ProjectsHomePageProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const projects = useProjects()
  const [isCreating, setCreating] = useState(false)
  const [createName, setCreateName] = useState('')
  const [renaming, setRenaming] = useState<Project | null>(null)
  const [renameName, setRenameName] = useState('')
  const [deleting, setDeleting] = useState<Project | null>(null)
  const [view, setView] = useLibraryView()

  const scope = useQuery({
    queryKey: ['pipeline-jobs', 'home'],
    queryFn: async () => {
      const projectList = await listProjects(token)
      const pairs = await Promise.all(
        projectList.map(async (project) => {
          const jobs = normalizeJobs(await listJobs(token, project.id)).filter((job) => job.projectId === project.id)
          return [project.id, jobs] as const
        }),
      )
      return {
        jobsByProject: Object.fromEntries(pairs) as Record<string, PipelineJob[]>,
      }
    },
    staleTime: 30_000,
  })

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['projects'] })
    await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', 'home'] })
  }

  const create = useMutation({
    mutationFn: (name: string) => createProject(token, name),
    onSuccess: async (project) => {
      await refresh()
      setCreateName('')
      setCreating(false)
      navigate(WORKSPACE_PATHS.project(project.id))
    },
  })

  const rename = useMutation({
    mutationFn: (project: Project) => updateProject(token, project.id, renameName),
    onSuccess: async () => {
      await refresh()
      setRenaming(null)
    },
  })

  const remove = useMutation({
    mutationFn: (project: Project) => deleteProject(token, project.id),
    onSuccess: async () => {
      await refresh()
      setDeleting(null)
    },
  })

  const rows = projects.data ?? []

  return (
    <WorkspacePage
      contentWidth={PAGE_CONTENT_WIDTH}
      header={
        <PageHeader
          title="Projects"
            description="Open a project to see its documents."
          end={<Button label="New project" variant="primary" onClick={() => setCreating(true)} />}
        />
      }
    >
      {projects.isPending ? <PendingLines /> : null}
      {projects.isError ? (
        <Banner status="error" title="Could not load projects" description={projectErrorMessage(projects.error)} />
      ) : null}
      {rows.length === 0 && !projects.isPending && !projects.isError ? (
        <EmptyState
          title="No projects yet"
          description="Create a project, then add one recording at a time."
          icon={<Icon icon={FolderIcon} size="lg" color={MUTED_ICON_COLOR} />}
          isCompact
        />
      ) : null}
      {rows.length > 0 ? (
        <VStack gap={WORKSPACE_GAP.group}>
          <HStack justify="end" vAlign="center">
            <LibraryViewControl value={view} onChange={setView} />
          </HStack>
          {view === 'grid' ? (
            <Grid columns={{ minWidth: 240, max: 4 }} gap={WORKSPACE_GAP.group} width="100%" aria-label="Projects">
              {rows.map((project) => {
                const files = projectFileCount(project.id, scope.data, scope.isPending)
                return (
                  <ProjectTile
                    key={project.id}
                    project={project}
                    files={files}
                    countsReady={!scope.isPending}
                    onRename={(item) => {
                      setRenameName(item.name)
                      setRenaming(item)
                    }}
                    onDelete={setDeleting}
                  />
                )
              })}
            </Grid>
          ) : (
            <List density="spacious" aria-label="Projects">
              {rows.map((project) => {
                const files = projectFileCount(project.id, scope.data, scope.isPending)
                const summary = projectSummary(files, !scope.isPending)
                return (
                  <ListItem
                    key={project.id}
                    href={WORKSPACE_PATHS.project(project.id)}
                    label={project.name}
                    description={summary ?? undefined}
                    startContent={<Icon icon={FolderIcon} size="sm" color={MUTED_ICON_COLOR} />}
                    endContent={
                      <MoreMenu
                        label={`Manage ${project.name}`}
                        size="sm"
                        alignment="end"
                        items={[
                          {
                            label: 'Rename',
                            onClick: () => {
                              setRenameName(project.name)
                              setRenaming(project)
                            },
                          },
                          { type: 'divider' },
                          { label: 'Delete', variant: 'destructive', onClick: () => setDeleting(project) },
                        ]}
                      />
                    }
                  />
                )
              })}
            </List>
          )}
        </VStack>
      ) : null}

      <Dialog isOpen={isCreating} onOpenChange={setCreating} purpose="form">
        <DialogHeader title="New project" subtitle="Jobs stay inside it." onOpenChange={setCreating} />
        <VStack gap={WORKSPACE_GAP.group}>
          {create.isError ? (
            <Banner status="error" title="Could not create the project" description={projectErrorMessage(create.error)} />
          ) : null}
          <TextInput label="Name" value={createName} onChange={setCreateName} />
          <HStack gap={WORKSPACE_GAP.local} justify="end">
            <Button label="Cancel" variant="secondary" onClick={() => setCreating(false)} />
            <Button
              label="Create"
              variant="primary"
              isDisabled={!createName.trim() || create.isPending}
              isLoading={create.isPending}
              onClick={() => create.mutate(createName)}
            />
          </HStack>
        </VStack>
      </Dialog>

      <Dialog isOpen={renaming !== null} onOpenChange={(open) => { if (!open) setRenaming(null) }} purpose="form">
        <DialogHeader title="Rename project" onOpenChange={(open) => { if (!open) setRenaming(null) }} />
        <VStack gap={WORKSPACE_GAP.group}>
          {rename.isError ? (
            <Banner status="error" title="Could not rename the project" description={projectErrorMessage(rename.error)} />
          ) : null}
          <TextInput label="Name" value={renameName} onChange={setRenameName} />
          <HStack gap={WORKSPACE_GAP.local} justify="end">
            <Button label="Cancel" variant="secondary" onClick={() => setRenaming(null)} />
            <Button
              label="Save"
              variant="primary"
              isDisabled={!renameName.trim() || rename.isPending}
              isLoading={rename.isPending}
              onClick={() => { if (renaming) rename.mutate(renaming) }}
            />
          </HStack>
        </VStack>
      </Dialog>

      <AlertDialog
        isOpen={deleting !== null}
        onOpenChange={(open) => { if (!open) setDeleting(null) }}
        title="Delete project"
        description={
          remove.isError
            ? projectErrorMessage(remove.error)
            : deleting
              ? `${deleting.name} and its place in the library go away. Its jobs are not moved to another project.`
              : 'This project goes away.'
        }
        actionLabel="Delete"
        isActionLoading={remove.isPending}
        onAction={() => { if (deleting) remove.mutate(deleting) }}
      />
    </WorkspacePage>
  )
}
