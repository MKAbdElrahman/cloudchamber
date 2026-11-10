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

The words of a 42 minute conversation, sectioned for a reader.

00:00

## Opening

The lead describes what a finished job should feel like: the recording is the source, and the page is the document.

00:18

## What they asked for

> Every job should show the source and the deliverable, and the deliverable should be something you can read.

02:40

## Close

Keep the definition and the correction. Leave the rest of the wording in this section for the reader who wants it.`,
  'pipeline-output.md': `# How a recording becomes a document

The transcript, sectioned, with the frames that carry the argument.

00:12

## Ingest

The source is a talk. CloudChamber keeps the words and drops the waiting: the pause before the first slide, the room tone, the aside that never lands.

00:42

## The argument

A pipeline is reliable when every job can show its source and its deliverable in one place. The slide on screen is the claim. The sentence under it is the proof.

02:10

## What to keep

Keep the diagram, the definition, and the moment the speaker corrects themselves. Leave the rest in the transcript for the reader who wants the full wording.`,
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

/** The one document for a job. A completed run without a demo row gets the reading sample. */
export function outputsForJob(jobId: string, jobStatus?: string): OutputFile[] {
  const linked = MOCK_OUTPUT_FILES.filter((file) => file.jobId === jobId)
  if (linked.length > 0) return linked.slice(0, 1)
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
  withMarkdown({
    id: 'out-md-2',
    name: 'pipeline-output.md',
    kind: 'markdown',
    dateUpdated: '2026-03-07T09:15:00.000Z',
    jobId: 'job-demo-02',
  }),
]
