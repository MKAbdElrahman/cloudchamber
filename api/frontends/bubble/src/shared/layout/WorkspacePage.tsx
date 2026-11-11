import type { ReactNode } from 'react'

import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout'

import { WORKSPACE_CONTENT_PADDING } from './contentSurface'

type WorkspacePageProps = {
  /** Header body. Wrapped in the single page header; omit on screens with no header. */
  header?: ReactNode
  /** Optional start panel (settings nav). */
  start?: ReactNode
  /**
   * Aligned measure for this page. Settings uses 640. With one panel the content
   * region still extends to the open edge; without a panel the children center
   * in that measure.
   */
  contentWidth?: number
  children: ReactNode
}

/**
 * Signed-in screen frame. Fills the shell's main region. Pages that are text
 * or settings pass contentWidth so the line length holds while the region resizes.
 */
export function WorkspacePage({ header, start, contentWidth, children }: WorkspacePageProps) {
  return (
    <Layout
      height="fill"
      padding={WORKSPACE_CONTENT_PADDING}
      contentWidth={contentWidth}
      start={start}
      header={header ? <LayoutHeader hasDivider>{header}</LayoutHeader> : undefined}
      content={<LayoutContent>{children}</LayoutContent>}
    />
  )
}
