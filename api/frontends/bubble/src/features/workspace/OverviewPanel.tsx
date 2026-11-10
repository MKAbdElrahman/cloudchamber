import { useNavigate } from 'react-router-dom'

import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout'
import { Text } from '@astryxdesign/core/Text'
import { Button } from '@astryxdesign/core/Button'
import { List, ListItem } from '@astryxdesign/core/List'
import { Icon } from '@astryxdesign/core/Icon'
import { Token } from '@astryxdesign/core/Token'
import { Timestamp } from '@astryxdesign/core/Timestamp'
import { DocumentIcon, DocumentTextIcon, PlayCircleIcon } from '@heroicons/react/24/outline'

import { PageHeader } from '@/shared/layout/PageHeader'
import { WORKSPACE_CONTENT_PADDING, WORKSPACE_CONTENT_WIDTH } from '@/shared/layout/contentSurface'
import { WORKSPACE_PATHS } from '@/shared/layout/workspacePaths'
import { META_TOKEN_COLOR, SHELL_ICON_COLOR } from '@/shared/theme/presentation'

import { MOCK_OUTPUT_FILES, type MockOutputFile } from './mockOutputs'

function outputIcon(file: MockOutputFile) {
  return file.kind === 'pdf' ? DocumentIcon : DocumentTextIcon
}

function kindLabel(kind: MockOutputFile['kind']) {
  return kind === 'pdf' ? 'PDF' : 'Markdown'
}

/** Studio home — spacious list, one primary path to the pipeline. */
export function OverviewPanel() {
  const navigate = useNavigate()

  return (
    <Layout
      height="fill"
      contentWidth={WORKSPACE_CONTENT_WIDTH}
      padding={WORKSPACE_CONTENT_PADDING}
    >
      <LayoutHeader hasDivider>
        <PageHeader
          title="Previous output"
          description="Open the pipeline to upload, run jobs, and monitor status."
          end={
            <Button
              label="Open pipeline"
              variant="primary"
              icon={<PlayCircleIcon />}
              onClick={() => navigate(WORKSPACE_PATHS.pipeline)}
            />
          }
        />
      </LayoutHeader>
      <LayoutContent>
        <List density="spacious" edge="flush" hasDividers aria-label="Previous outputs">
          {MOCK_OUTPUT_FILES.map((file) => (
            <ListItem
              key={file.id}
              label={file.name}
              description={
                <Text type="supporting">
                  {file.jobId} · <Timestamp value={file.dateUpdated} format="relative" />
                </Text>
              }
              startContent={<Icon icon={outputIcon(file)} size="sm" color={SHELL_ICON_COLOR} />}
              endContent={<Token label={kindLabel(file.kind)} size="sm" color={META_TOKEN_COLOR} />}
            />
          ))}
        </List>
      </LayoutContent>
    </Layout>
  )
}
