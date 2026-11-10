import { VStack } from '@astryxdesign/core/Layout'

import { WORKSPACE_GAP } from '@/shared/spacing'

import { JobOutputFile } from './JobOutputFile'
import type { OutputFile } from './types'

type JobOutputsPanelProps = {
  files: OutputFile[]
}

export function JobOutputsPanel({ files }: JobOutputsPanelProps) {
  return (
    <VStack gap={WORKSPACE_GAP.section} aria-label="Job outputs">
      {files.map((file) => (
        <JobOutputFile key={file.id} file={file} />
      ))}
    </VStack>
  )
}
