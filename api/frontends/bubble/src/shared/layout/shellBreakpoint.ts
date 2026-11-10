import { DEFAULT_WIDTH_BREAKPOINTS } from '@astryxdesign/core/theme'

/**
 * Settings start panel drops here and the section list stacks above the pane.
 * This is wider than the shell's mobile nav (md / 768): between 768 and 1024
 * the side nav stays, and only the second region is removed.
 */
export const SHELL_NARROW_QUERY = `(width < ${DEFAULT_WIDTH_BREAKPOINTS.lg}px)`
