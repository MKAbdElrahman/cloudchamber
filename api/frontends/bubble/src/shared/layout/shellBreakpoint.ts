import { DEFAULT_WIDTH_BREAKPOINTS } from '@astryxdesign/core/theme'

/**
 * Second region (settings list, document outline) swaps into the content flow
 * below lg. The shell's own nav swaps at md, so between 768 and 1024 the side
 * nav stays and only this region is removed.
 */
export const SHELL_NARROW_QUERY = `(width < ${DEFAULT_WIDTH_BREAKPOINTS.lg}px)`
