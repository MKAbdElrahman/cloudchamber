import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { HStack, VStack } from '@astryxdesign/core/Layout'
import { AlertDialog } from '@astryxdesign/core/AlertDialog'
import { Banner } from '@astryxdesign/core/Banner'
import { Button } from '@astryxdesign/core/Button'
import { ContextMenu, type ContextMenuOption } from '@astryxdesign/core/ContextMenu'
import { Dialog, DialogHeader } from '@astryxdesign/core/Dialog'
import { EmptyState } from '@astryxdesign/core/EmptyState'
import { List, ListItem } from '@astryxdesign/core/List'
import { MoreMenu } from '@astryxdesign/core/MoreMenu'
import { TextInput } from '@astryxdesign/core/TextInput'
import { useToast } from '@astryxdesign/core/Toast'
import {
  createProject,
  deleteProject,
  listProjects,
  projectErrorMessage,
  updateProject,
  useProjects,
  type Project,
} from '@/features/projects'
import { jobNeedsOpen, listJobs, normalizeJobs, openUnsettledJobs, type PipelineJob } from '@/features/pipeline'
import { filesFromJob } from '@/features/outputs'
import { PAGE_CONTENT_WIDTH, PageHeader, PendingLines, WorkspacePage, WORKSPACE_PATHS } from '@/shared/layout'
import { WORKSPACE_GAP } from '@/shared/spacing'

type ProjectsHomePageProps = {
  token: string
}

function countLabel(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`
}

type ProjectTileProps = {
  project: Project
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

function projectActions(
  project: Project,
  onRename: (project: Project) => void,
  onDelete: (project: Project) => void,
): ContextMenuOption[] {
  return [
    { label: 'Rename', onClick: () => onRename(project) },
    { type: 'divider' },
    { label: 'Delete', variant: 'destructive', onClick: () => onDelete(project) },
  ]
}

/** Right-click opens the same actions at the pointer. A long press opens them on touch. */
function ProjectMenu({
  project,
  onRename,
  onDelete,
  role,
  children,
}: ProjectTileProps & { role?: 'presentation'; children: ReactNode }) {
  return (
    <ContextMenu
      presentation="adaptive"
      label={`Manage ${project.name}`}
      role={role}
      items={projectActions(project, onRename, onDelete)}
    >
      {children}
    </ContextMenu>
  )
}

/** Visible overflow. The corner of a tile, or the end of a row. */
function ProjectMoreMenu({
  project,
  onRename,
  onDelete,
}: ProjectTileProps) {
  return (
    <MoreMenu
      label={`Manage ${project.name}`}
      size="sm"
      alignment="end"
      presentation="adaptive"
      items={projectActions(project, onRename, onDelete)}
    />
  )
}

/** Home — each project is a row you open, rename, or delete. */
export function ProjectsHomePage({ token }: ProjectsHomePageProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const toast = useToast()
  const projects = useProjects()
  const [isCreating, setCreating] = useState(false)
  const [createName, setCreateName] = useState('')
  const [renaming, setRenaming] = useState<Project | null>(null)
  const [renameName, setRenameName] = useState('')
  const [deleting, setDeleting] = useState<Project | null>(null)

  const scope = useQuery({
    queryKey: ['pipeline-jobs', 'home'],
    queryFn: async () => {
      const projectList = await listProjects(token)
      const pairs = await Promise.all(
        projectList.map(async (project) => {
          const listed = normalizeJobs(await listJobs(token, project.id)).filter((job) => job.projectId === project.id)
          const jobs = await openUnsettledJobs(token, listed)
          return [project.id, jobs] as const
        }),
      )
      return {
        jobsByProject: Object.fromEntries(pairs) as Record<string, PipelineJob[]>,
      }
    },
    staleTime: 30_000,
    refetchInterval: (query) =>
      Object.values(query.state.data?.jobsByProject ?? {}).some((jobs) => jobs.some(jobNeedsOpen)) ? 3_000 : false,
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
    onSuccess: async (_removed, project) => {
      await refresh()
      setDeleting(null)
      toast({ body: `Deleted ${project.name}` })
    },
  })

  const rows = projects.data ?? []
  const projectsEmpty = rows.length === 0 && !projects.isPending && !projects.isError

  return (
    <WorkspacePage
      contentWidth={PAGE_CONTENT_WIDTH}
      header={
        <PageHeader
          title="Projects"
          end={
            projectsEmpty ? undefined : <Button label="New project" variant="primary" onClick={() => setCreating(true)} />
          }
        />
      }
    >
      {projects.isError ? (
        <Banner status="error" title="Could not load projects" description={projectErrorMessage(projects.error)} />
      ) : null}
      {rows.length === 0 && !projects.isPending && !projects.isError ? (
        <EmptyState
          title="No projects yet"
          isCompact
          actions={<Button label="New project" variant="primary" onClick={() => setCreating(true)} />}
        />
      ) : null}
      {projects.isPending || rows.length > 0 ? (
        <>
          {projects.isPending && rows.length === 0 ? <PendingLines variant="rows" /> : null}
          {rows.length > 0 ? (
            <List density="spacious" aria-label="Projects">
              {rows.map((project) => {
                const files = projectFileCount(project.id, scope.data, scope.isPending)
                const summary = projectSummary(files, !scope.isPending)
                const openRename = (item: Project) => {
                  setRenameName(item.name)
                  setRenaming(item)
                }
                return (
                  <ProjectMenu
                    key={project.id}
                    project={project}
                    onRename={openRename}
                    onDelete={setDeleting}
                    role="presentation"
                  >
                    <ListItem
                      href={WORKSPACE_PATHS.project(project.id)}
                      label={project.name}
                      description={summary ?? undefined}
                      endContent={<ProjectMoreMenu project={project} onRename={openRename} onDelete={setDeleting} />}
                    />
                  </ProjectMenu>
                )
              })}
            </List>
          ) : null}
        </>
      ) : null}

      <Dialog isOpen={isCreating} onOpenChange={setCreating} purpose="form">
        <DialogHeader title="New project" onOpenChange={setCreating} />
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
              ? deleting.name
              : 'This project goes away.'
        }
        actionLabel="Delete"
        isActionLoading={remove.isPending}
        onAction={() => { if (deleting) remove.mutate(deleting) }}
      />
    </WorkspacePage>
  )
}
