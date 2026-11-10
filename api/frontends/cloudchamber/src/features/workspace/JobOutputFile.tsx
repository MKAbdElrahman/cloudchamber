import { useState } from 'react'
import { DocumentIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { AspectRatio } from '@astryxdesign/core/AspectRatio'
import { Card } from '@astryxdesign/core/Card'
import { Icon } from '@astryxdesign/core/Icon'
import { MetadataList, MetadataListItem } from '@astryxdesign/core/MetadataList'
import { Text } from '@astryxdesign/core/Text'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { Token } from '@astryxdesign/core/Token'

import { InlineMediaPlayer } from '@/features/upload/inlineMediaPlayer'
import { WORKSPACE_CARD, WORKSPACE_GAP } from '@/shared/layout/contentSurface'
import { META_TOKEN_COLOR } from '@/shared/theme/presentation'

import type { MockOutputFile } from './mockOutputs'
import {
  mockMarkdownPreview,
  outputMediaKindForFile,
  playbackUrlForOutput,
} from './outputMedia'

function typeLabel(kind: MockOutputFile['kind']): string {
  switch (kind) {
    case 'video':
      return 'Video'
    case 'audio':
      return 'Audio'
    case 'pdf':
      return 'PDF'
    default:
      return 'Markdown'
  }
}

type JobOutputFileProps = {
  file: MockOutputFile
}

/** Single job output — inline preview (media, PDF, or markdown), not a list row link. */
export function JobOutputFile({ file }: JobOutputFileProps) {
  const [pdfError, setPdfError] = useState(false)
  const playbackUrl = playbackUrlForOutput(file)
  const mediaKind = outputMediaKindForFile(file)

  return (
    <Card padding={4} {...WORKSPACE_CARD}>
      <VStack gap={WORKSPACE_GAP.group}>
        <MetadataList>
          <MetadataListItem label="File">{file.name}</MetadataListItem>
          <MetadataListItem label="Type">
            <Token label={typeLabel(file.kind)} size="sm" color={META_TOKEN_COLOR} />
          </MetadataListItem>
          <MetadataListItem label="Updated">
            <Timestamp value={file.dateUpdated} format="relative" />
          </MetadataListItem>
        </MetadataList>

        {file.kind === 'video' || file.kind === 'audio' ? (
          <InlineMediaPlayer playbackUrl={playbackUrl} kind={mediaKind} />
        ) : null}

        {file.kind === 'pdf' ? (
          <>
            {playbackUrl && !pdfError ? (
              <AspectRatio ratio={4 / 3} fit="contain">
                <iframe
                  title={file.name}
                  src={playbackUrl}
                  onError={() => setPdfError(true)}
                  style={{ width: '100%', height: '100%', border: 'none', background: 'var(--color-background-surface)' }}
                />
              </AspectRatio>
            ) : (
              <AspectRatio ratio={4 / 3} fit="cover">
                <VStack hAlign="center" vAlign="center" gap={2} width="100%" height="100%">
                  <Icon icon={DocumentIcon} size="lg" color="secondary" />
                  <Text type="supporting" color="secondary" size="sm">
                    PDF preview will load when the file is available in storage.
                  </Text>
                </VStack>
              </AspectRatio>
            )}
          </>
        ) : null}

        {file.kind === 'markdown' ? (
          <Card padding={4} variant="muted">
            <Text type="supporting" size="sm" style={{ whiteSpace: 'pre-wrap', fontFamily: 'var(--font-family-mono, monospace)' }}>
              {mockMarkdownPreview(file)}
            </Text>
          </Card>
        ) : null}
      </VStack>
    </Card>
  )
}
