import { VStack } from '@astryxdesign/core/Layout'

import { WORKSPACE_GAP } from '@/shared/layout/contentSurface'

import { JobOutputFile } from './JobOutputFile'
import type { MockOutputFile } from './mockOutputs'

type JobOutputsPanelProps = {
  files: MockOutputFile[]
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
