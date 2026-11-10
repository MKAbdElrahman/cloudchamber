import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderIcon } from '@heroicons/react/24/outline'

import { Grid } from '@astryxdesign/core/Grid'
import { HStack, VStack } from '@astryxdesign/core/Layout'
import { AlertDialog } from '@astryxdesign/core/AlertDialog'
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
  claimUnassignedJobs,
  createProject,
  DEFAULT_PROJECT_ID,
  deleteProject,
  listProjectJobIds,
  listProjects,
  updateProject,
  useProjects,
  type Project,
} from '@/features/projects'
import { listJobs, normalizeJobs } from '@/features/pipeline'
import { PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'
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
  jobs: number
  countsReady: boolean
  onRename: (project: Project) => void
  onDelete: (project: Project) => void
}

function projectSummary(jobs: number, countsReady: boolean): string | null {
  if (!countsReady) return null
  if (jobs === 0) return 'No jobs yet.'
  return `${countLabel(jobs, 'job', 'jobs')}.`
}

type ProjectScope = {
  jobs: { id: string }[]
  jobIds: Record<string, string[]>
}

function projectJobCount(
  projectId: string,
  scope: ProjectScope | undefined,
  countsPending: boolean,
): number {
  if (countsPending || !scope) return 0
  const jobIds = new Set(scope.jobIds[projectId] ?? [])
  return scope.jobs.filter((job) => jobIds.has(job.id)).length
}

/** One project card. The folder mark names it; the card opens it. */
function ProjectTile({ project, jobs, countsReady, onRename, onDelete }: ProjectTileProps) {
  const summary = projectSummary(jobs, countsReady)

  return (
    <ClickableCard
      label={`Open ${project.name}`}
      href={WORKSPACE_PATHS.project(project.id)}
      padding={0}
      elevation="low"
      height="100%"
    >
      <VStack gap={0} height="100%">
        <HStack gap={3} padding={4} justify="between" vAlign="center">
          <HStack gap={3} vAlign="center">
            <Icon icon={FolderIcon} size="lg" color="secondary" />
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
        <VStack padding={4}>
          <Text type="body" color="secondary">
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
      const jobList = normalizeJobs(await listJobs(token))
      await claimUnassignedJobs(
        jobList.map((job) => job.id),
        DEFAULT_PROJECT_ID,
      )
      const projectList = await listProjects()
      const pairs = await Promise.all(
        projectList.map(async (project) => [project.id, await listProjectJobIds(project.id)] as const),
      )
      return {
        jobs: jobList,
        jobIds: Object.fromEntries(pairs) as Record<string, string[]>,
      }
    },
    staleTime: 30_000,
  })

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['projects'] })
    await queryClient.invalidateQueries({ queryKey: ['pipeline-jobs', 'home'] })
  }

  const create = useMutation({
    mutationFn: createProject,
    onSuccess: async (project) => {
      await refresh()
      setCreateName('')
      setCreating(false)
      navigate(WORKSPACE_PATHS.project(project.id))
    },
  })

  const rename = useMutation({
    mutationFn: (project: Project) => updateProject(project.id, renameName),
    onSuccess: async () => {
      await refresh()
      setRenaming(null)
    },
  })

  const remove = useMutation({
    mutationFn: (project: Project) => deleteProject(project.id),
    onSuccess: async () => {
      await refresh()
      setDeleting(null)
    },
  })

  const rows = projects.data ?? []

  return (
    <WorkspacePage
      header={
        <PageHeader
          title="Projects"
          description="Open a project to see its jobs."
          end={<Button label="New project" variant="primary" onClick={() => setCreating(true)} />}
        />
      }
    >
      {projects.isPending ? <PendingLines /> : null}
      {rows.length === 0 && !projects.isPending ? (
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
            <Grid columns={{ minWidth: 240, max: 4 }} gap={4} width="100%" aria-label="Projects">
              {rows.map((project) => {
                const jobs = projectJobCount(project.id, scope.data, scope.isPending)
                return (
                  <ProjectTile
                    key={project.id}
                    project={project}
                    jobs={jobs}
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
            <List density="spacious" edge="flush" hasDividers aria-label="Projects">
              {rows.map((project) => {
                const jobs = projectJobCount(project.id, scope.data, scope.isPending)
                const summary = projectSummary(jobs, !scope.isPending)
                return (
                  <ListItem
                    key={project.id}
                    href={WORKSPACE_PATHS.project(project.id)}
                    label={project.name}
                    description={summary ?? undefined}
                    startContent={<Icon icon={FolderIcon} size="sm" color="secondary" />}
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
          deleting
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
