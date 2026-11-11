import { useQuery } from '@tanstack/react-query'

import { listProjectFiles } from '@/features/projects'

/** Files stored in one project, newest first. */
export function useProjectFiles(token: string, projectId: string) {
  return useQuery({
    queryKey: ['project-files', projectId],
    queryFn: () => listProjectFiles(token, projectId),
    enabled: Boolean(token) && Boolean(projectId),
    staleTime: 30_000,
    refetchOnMount: 'always',
  })
}
