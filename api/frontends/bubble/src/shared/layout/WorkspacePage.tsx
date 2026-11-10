import type { CSSProperties, ReactNode } from 'react'

import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout'

import { WORKSPACE_CONTENT_PADDING, WORKSPACE_CONTENT_WIDTH } from './contentSurface'

type WorkspacePageProps = {
  /** Header body. Wrapped in the single page header; omit on screens with no header. */
  header?: ReactNode
  /** Optional start panel (settings nav). */
  start?: ReactNode
  children: ReactNode
  style?: CSSProperties
}

/** Signed-in screen frame. Width, padding, and the header rule live here. */
export function WorkspacePage({ header, start, children, style }: WorkspacePageProps) {
  return (
    <Layout
      height="fill"
      contentWidth={WORKSPACE_CONTENT_WIDTH}
      padding={WORKSPACE_CONTENT_PADDING}
      style={style}
      start={start}
      header={header ? <LayoutHeader hasDivider>{header}</LayoutHeader> : undefined}
      content={<LayoutContent>{children}</LayoutContent>}
    />
  )
}
