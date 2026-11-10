export type OutputFileKind = 'pdf' | 'markdown' | 'video' | 'audio'

export function outputKindFromFileName(name: string): OutputFileKind {
  const lower = name.toLowerCase()
  if (/\.(mp4|webm|mov|m4v|mkv|avi|ogv)$/.test(lower)) return 'video'
  if (/\.(mp3|wav|ogg|m4a|aac|flac)$/.test(lower)) return 'audio'
  if (lower.endsWith('.pdf')) return 'pdf'
  return 'markdown'
}

export type MockOutputFile = {
  id: string
  name: string
  kind: OutputFileKind
  /** ISO timestamp for display */
  dateUpdated: string
  jobId: string
}

/** Deliverables linked to a job — demo rows first, then a generic placeholder when completed. */
export function outputsForJob(jobId: string, jobStatus?: string): MockOutputFile[] {
  const linked = MOCK_OUTPUT_FILES.filter((file) => file.jobId === jobId)
  if (linked.length > 0) return linked
  if (jobStatus !== 'completed') return []
  const name = 'pipeline-output.md'
  return [
    {
      id: `out-mock-${jobId}`,
      name,
      kind: outputKindFromFileName(name),
      dateUpdated: new Date().toISOString(),
      jobId,
    },
  ]
}

export const MOCK_OUTPUT_FILES: MockOutputFile[] = [
  {
    id: 'out-md-1',
    name: 'interview-transcript.md',
    kind: 'markdown',
    dateUpdated: '2026-03-08T14:22:00.000Z',
    jobId: 'job-demo-01',
  },
  {
    id: 'out-vid-1',
    name: 'highlights-reel.mp4',
    kind: 'video',
    dateUpdated: '2026-03-08T14:25:00.000Z',
    jobId: 'job-demo-01',
  },
  {
    id: 'out-pdf-1',
    name: 'structured-summary.pdf',
    kind: 'pdf',
    dateUpdated: '2026-03-07T09:15:00.000Z',
    jobId: 'job-demo-02',
  },
]
