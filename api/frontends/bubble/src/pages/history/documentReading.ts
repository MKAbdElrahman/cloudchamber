import { parseMarkdown, type InlineNode, type SourceRange } from '@astryxdesign/core/Markdown/parser'

const TIME_LINE = /^\d{1,2}:\d{2}$/

export type ReadingDocument = {
  title?: string
  dek?: string
  body: string
  timeByHeading: ReadonlyMap<string, string>
}

function inlineText(nodes: readonly InlineNode[]): string {
  let text = ''
  for (const node of nodes) {
    switch (node.type) {
      case 'text':
      case 'code':
        text += node.content
        break
      case 'bold':
      case 'italic':
      case 'strikethrough':
      case 'link':
        text += inlineText(node.children)
        break
      case 'image':
        text += node.alt
        break
      case 'break':
      case 'citation':
        break
      default:
        break
    }
  }
  return text.trim()
}

function withoutRanges(source: string, ranges: readonly SourceRange[]): string {
  const ordered = [...ranges].sort((a, b) => a.start - b.start)
  let cursor = 0
  let body = ''
  for (const range of ordered) {
    body += source.slice(cursor, range.start)
    cursor = range.end
  }
  body += source.slice(cursor)
  return body.replace(/\n{3,}/g, '\n\n').trim()
}

/**
 * Lifts the title, the line under it, and each section time out of the source.
 * The remaining string is what Markdown and Outline both render.
 */
export function readingDocument(markdown: string): ReadingDocument {
  const blocks = parseMarkdown(markdown, { sourceRanges: true })
  const dropped: SourceRange[] = []
  const timeByHeading = new Map<string, string>()
  let title: string | undefined
  let dek: string | undefined

  const first = blocks[0]
  if (first?.type === 'heading' && first.level === 1 && first.range) {
    title = inlineText(first.children)
    dropped.push(first.range)
    const second = blocks[1]
    if (second?.type === 'paragraph' && second.range) {
      const text = inlineText(second.children)
      if (!TIME_LINE.test(text)) {
        dek = text
        dropped.push(second.range)
      }
    }
  }

  for (let index = 0; index < blocks.length; index += 1) {
    const block = blocks[index]
    const next = blocks[index + 1]
    if (block?.type !== 'paragraph' || next?.type !== 'heading' || !block.range) continue
    const time = inlineText(block.children)
    if (!TIME_LINE.test(time)) continue
    timeByHeading.set(inlineText(next.children), time)
    dropped.push(block.range)
  }

  return {
    title,
    dek,
    body: withoutRanges(markdown, dropped),
    timeByHeading,
  }
}
