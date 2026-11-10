import type { MouseEvent } from 'react'
import { FilmIcon, MusicalNoteIcon, QueueListIcon } from '@heroicons/react/24/outline'

import { HStack } from '@astryxdesign/core/Layout'
import { Icon } from '@astryxdesign/core/Icon'
import { IconButton } from '@astryxdesign/core/IconButton'
import type { SourceMediaKind } from '@/shared/media'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { jobPhase } from './jobStatus'

export type JobPeekTarget = 'video' | 'pipeline'

type JobOpenIconsProps = {
  kind: SourceMediaKind
  /** Set when this job has a conversion to open. */
  hasPipeline?: boolean
  onOpen: (target: JobPeekTarget) => void
  status?: string
}

function openLabel(kind: SourceMediaKind): string {
  switch (kind) {
    case 'audio':
      return 'Audio'
    case 'video':
      return 'Video'
    default:
      return 'Recording'
  }
}

/** Icons that open the recording or the pipeline over the current page. */
export function JobOpenIcons({ kind, hasPipeline = false, onOpen, status }: JobOpenIconsProps) {
  const mediaIcon = kind === 'audio' ? MusicalNoteIcon : FilmIcon
  const label = openLabel(kind)

  function press(target: JobPeekTarget) {
    return (event: MouseEvent) => {
      event.preventDefault()
      event.stopPropagation()
      onOpen(target)
    }
  }

  return (
    <HStack gap={WORKSPACE_GAP.local} vAlign="center">
      <IconButton
        label={label}
        tooltip={`Open ${label.toLowerCase()}`}
        variant="ghost"
        size="sm"
        icon={<Icon icon={mediaIcon} />}
        onClick={press('video')}
      />
      {hasPipeline ? (
        <IconButton
          label="Pipeline"
          tooltip="Open pipeline"
          variant="ghost"
          size="sm"
          isLoading={jobPhase(status) === 'running'}
          icon={<Icon icon={QueueListIcon} />}
          onClick={press('pipeline')}
        />
      ) : null}
    </HStack>
  )
}
