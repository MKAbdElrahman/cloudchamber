import { useState } from 'react'
import { FilmIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { AspectRatio } from '@astryxdesign/core/AspectRatio'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'

import { WORKSPACE_GAP } from '@/shared/spacing'
import { MUTED_ICON_COLOR } from '@/shared/theme'

import type { SourceMediaKind } from './sourceMedia'

type InlineMediaPlayerProps = {
  playbackUrl: string | null
  kind: SourceMediaKind
  fileName?: string
}

export function InlineMediaPlayer({ playbackUrl, kind, fileName }: InlineMediaPlayerProps) {
  const [playbackError, setPlaybackError] = useState(false)

  if (playbackUrl && !playbackError) {
    return (
      <AspectRatio ratio={16 / 9} fit="contain">
        {kind === 'audio' ? (
          <audio
            controls
            preload="metadata"
            src={playbackUrl}
            onError={() => setPlaybackError(true)}
          >
            Your browser does not support audio playback.
          </audio>
        ) : (
          <video
            controls
            playsInline
            preload="metadata"
            src={playbackUrl}
            onError={() => setPlaybackError(true)}
          >
            Your browser does not support video playback.
          </video>
        )}
      </AspectRatio>
    )
  }

  return (
    <AspectRatio ratio={16 / 9} fit="cover">
      <VStack hAlign="center" vAlign="center" gap={WORKSPACE_GAP.local} width="100%" height="100%">
        <Icon icon={FilmIcon} size="lg" color={MUTED_ICON_COLOR} />
        <Text type="supporting">
          {fileName ? `${fileName} cannot be played from here.` : 'This recording cannot be played from here.'}
        </Text>
      </VStack>
    </AspectRatio>
  )
}
