import { TextInput } from '@astryxdesign/core/TextInput'

type SettingsSectionSearchProps = {
  query: string
  onQueryChange: (value: string) => void
}

export function SettingsSectionSearch({ query, onQueryChange }: SettingsSectionSearchProps) {
  return (
    <TextInput
      label="Search settings"
      isLabelHidden
      placeholder="Search settings"
      value={query}
      onChange={onQueryChange}
      size="sm"
    />
  )
}
