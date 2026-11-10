import { env } from '@/shared/config/env'

import { sourceMediaKind, type SourceMediaKind } from '@/features/upload'

import type { MockOutputFile } from './mockOutputs'

export function outputMediaKindForFile(file: MockOutputFile): SourceMediaKind {
  if (file.kind === 'video' || file.kind === 'audio') return file.kind
  return sourceMediaKind(file.name)
}

export function playbackUrlForOutput(file: MockOutputFile): string | null {
  const objectKey = `outputs/${file.jobId}/${file.name}`
  const base = env.tusdEndpoint.endsWith('/') ? env.tusdEndpoint : `${env.tusdEndpoint}/`
  const encoded = objectKey.split('/').map(encodeURIComponent).join('/')
  return new URL(encoded, base).href
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

export function mockMarkdownPreview(file: MockOutputFile): string {
  return (
    MARKDOWN_PREVIEWS[file.name] ??
  `# ${file.name}

Placeholder markdown output until artifact storage is wired to the API.`
  )
}
