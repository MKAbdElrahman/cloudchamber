import type { OutputFile, OutputFileKind } from '@/features/outputs'

function outputKindFromFileName(name: string): OutputFileKind {
  const lower = name.toLowerCase()
  if (/\.(mp4|webm|mov|m4v|mkv|avi|ogv)$/.test(lower)) return 'video'
  if (/\.(mp3|wav|ogg|m4a|aac|flac)$/.test(lower)) return 'audio'
  if (lower.endsWith('.pdf')) return 'pdf'
  return 'markdown'
}

const MARKDOWN_PREVIEWS: Record<string, string> = {
  'interview-transcript.md': `# Interview transcript

**Participant:** Engineering lead  
**Duration:** 42 min

## Summary

Discussion covered pipeline reliability, tus upload verification, and artifact handoff.

## Excerpt

> We want every job to show the source recording and generated deliverables in one place.`,
  'pipeline-output.md': `# Pipeline output

Mock deliverable for a completed run.

## Sections

1. Ingest validation  
2. Transcription (placeholder)  
3. Structured summary (placeholder)`,
}

function withMarkdown(file: OutputFile): OutputFile {
  if (file.kind !== 'markdown') return file
  return {
    ...file,
    markdown:
      MARKDOWN_PREVIEWS[file.name] ??
      `# ${file.name}

Placeholder markdown output until artifact storage is wired to the API.`,
  }
}

/** Deliverables linked to a job — demo rows first, then a generic placeholder when completed. */
export function outputsForJob(jobId: string, jobStatus?: string): OutputFile[] {
  const linked = MOCK_OUTPUT_FILES.filter((file) => file.jobId === jobId)
  if (linked.length > 0) return linked
  if (jobStatus !== 'completed') return []
  const name = 'pipeline-output.md'
  return [
    withMarkdown({
      id: `out-mock-${jobId}`,
      name,
      kind: outputKindFromFileName(name),
      dateUpdated: new Date().toISOString(),
      jobId,
    }),
  ]
}

export const MOCK_OUTPUT_FILES: OutputFile[] = [
  withMarkdown({
    id: 'out-md-1',
    name: 'interview-transcript.md',
    kind: 'markdown',
    dateUpdated: '2026-03-08T14:22:00.000Z',
    jobId: 'job-demo-01',
  }),
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
