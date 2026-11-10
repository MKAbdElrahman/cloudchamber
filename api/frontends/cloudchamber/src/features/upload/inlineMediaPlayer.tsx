import { useState } from 'react'
import { FilmIcon } from '@heroicons/react/24/outline'

import { VStack } from '@astryxdesign/core/Layout'
import { AspectRatio } from '@astryxdesign/core/AspectRatio'
import { Icon } from '@astryxdesign/core/Icon'
import { Text } from '@astryxdesign/core/Text'

import type { SourceMediaKind } from './sourceMedia'

type InlineMediaPlayerProps = {
  playbackUrl: string | null
  kind: SourceMediaKind
}

export function InlineMediaPlayer({ playbackUrl, kind }: InlineMediaPlayerProps) {
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
            style={{ width: '100%' }}
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
            style={{
              width: '100%',
              height: '100%',
              objectFit: 'contain',
              background: 'var(--color-background-surface)',
            }}
          >
            Your browser does not support video playback.
          </video>
        )}
      </AspectRatio>
    )
  }

  return (
    <AspectRatio ratio={16 / 9} fit="cover">
      <VStack hAlign="center" vAlign="center" gap={2} width="100%" height="100%">
        <Icon icon={FilmIcon} size="lg" color="secondary" />
        <Text type="supporting" color="secondary" size="sm">
          {playbackError
            ? 'Preview is not available from this environment.'
            : 'Preview will appear when the file URL can be resolved.'}
        </Text>
      </VStack>
    </AspectRatio>
  )
}
