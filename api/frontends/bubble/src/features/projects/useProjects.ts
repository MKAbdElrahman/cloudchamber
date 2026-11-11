import { useQuery } from '@tanstack/react-query'

import { useSession } from '@/shared/auth'

import { listProjects } from './api'

export function useProjects() {
  const { token } = useSession()
  return useQuery({
    queryKey: ['projects', token],
    queryFn: () => listProjects(token ?? ''),
    enabled: Boolean(token),
  })
}

export function useProject(projectId: string) {
  const projects = useProjects()
  return projects.data?.find((project) => project.id === projectId)
}
