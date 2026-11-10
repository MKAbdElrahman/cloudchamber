import type { ReactNode } from 'react'

import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout'

import { WORKSPACE_CONTENT_PADDING } from './contentSurface'

type WorkspacePageProps = {
  /** Header body. Wrapped in the single page header; omit on screens with no header. */
  header?: ReactNode
  /** Optional start panel (settings nav). */
  start?: ReactNode
  children: ReactNode
}

/**
 * Signed-in screen frame. Fills the shell's main region (no content-width cap)
 * so the page tracks the viewport from a phone up to a wide monitor.
 */
export function WorkspacePage({ header, start, children }: WorkspacePageProps) {
  return (
    <Layout
      height="fill"
      padding={WORKSPACE_CONTENT_PADDING}
      start={start}
      header={header ? <LayoutHeader hasDivider>{header}</LayoutHeader> : undefined}
      content={<LayoutContent>{children}</LayoutContent>}
    />
  )
}
