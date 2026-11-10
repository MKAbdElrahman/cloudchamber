import { Banner } from '@astryxdesign/core/Banner'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Token } from '@astryxdesign/core/Token'
import { VStack } from '@astryxdesign/core/Layout'

import { WORKSPACE_GAP } from '@/shared/layout/contentSurface'
import { META_TOKEN_COLOR } from '@/shared/theme/presentation'

import { InlineMediaPlayer } from './inlineMediaPlayer'
import {
  fileNameFromSourceRef,
  playbackUrlFromSourceRef,
  sourceMediaKind,
  type SourceMediaKind,
} from './sourceMedia'

function kindLabel(kind: SourceMediaKind): string {
  switch (kind) {
    case 'video':
      return 'Video'
    case 'audio':
      return 'Audio'
    default:
      return 'Media'
  }
}

type JobInputMediaProps = {
  sourceRef: string
}

/** Job input recording — filename and inline player, not a bare sourceRef link. */
export function JobInputMedia({ sourceRef }: JobInputMediaProps) {
  const fileName = fileNameFromSourceRef(sourceRef)
  const kind = sourceMediaKind(sourceRef)
  const playbackUrl = playbackUrlFromSourceRef(sourceRef)

  return (
    <VStack gap={WORKSPACE_GAP.group}>
      <MetadataList>
        <MetadataListItem label="File">{fileName}</MetadataListItem>
        <MetadataListItem label="Type">
          <Token label={kindLabel(kind)} size="sm" color={META_TOKEN_COLOR} />
        </MetadataListItem>
      </MetadataList>

      <InlineMediaPlayer playbackUrl={playbackUrl} kind={kind} />

      {!playbackUrl ? (
        <Banner
          status="info"
          title="Could not load preview"
          description="The file is still referenced for this job; download or playback may require API access."
        />
      ) : null}
    </VStack>
  )
}
