export type DocumentHeading = {
  kind: 'heading'
  level: 1 | 2 | 3
  text: string
  id: string
  time?: string
}

export type DocumentBlock =
  | DocumentHeading
  | { kind: 'paragraph'; text: string }
  | { kind: 'quote'; text: string }

const TIME_LINE = /^(\d{1,2}:\d{2})$/

function slug(text: string, used: Set<string>): string {
  const base = text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'section'
  let id = base
  let n = 2
  while (used.has(id)) {
    id = `${base}-${n}`
    n += 1
  }
  used.add(id)
  return id
}

function plain(text: string): string {
  return text.replace(/\*\*/g, '').trim()
}

/** A small reading parse of the one markdown a job produces. */
export function parseDocument(markdown: string): DocumentBlock[] {
  const used = new Set<string>()
  const blocks: DocumentBlock[] = []
  let pendingTime: string | undefined
  let paragraph: string[] = []

  function flushParagraph() {
    const text = plain(paragraph.join(' '))
    paragraph = []
    if (text) blocks.push({ kind: 'paragraph', text })
  }

  for (const raw of markdown.replace(/\r\n/g, '\n').split('\n')) {
    const line = raw.trim()
    if (!line) {
      flushParagraph()
      continue
    }
    const time = TIME_LINE.exec(line)
    if (time) {
      flushParagraph()
      pendingTime = time[1]
      continue
    }
    const heading = /^(#{1,3})\s+(.+)$/.exec(line)
    if (heading) {
      flushParagraph()
      const text = plain(heading[2])
      blocks.push({
        kind: 'heading',
        level: heading[1].length as 1 | 2 | 3,
        text,
        id: slug(text, used),
        time: pendingTime,
      })
      pendingTime = undefined
      continue
    }
    if (line.startsWith('>')) {
      flushParagraph()
      blocks.push({ kind: 'quote', text: plain(line.replace(/^>\s?/, '')) })
      continue
    }
    paragraph.push(line)
  }
  flushParagraph()
  return blocks
}
