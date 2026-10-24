import { expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { App, ConfigProvider } from 'antd'
import { antdTheme } from '@/lib/antd-theme'
import GameScenesEditor from './GameScenesEditor'
import type { GameProjectAsset, GameProjectCharacter, GameProjectScene } from '@/api/gameProjects'

const assets: GameProjectAsset[] = [
  { id: 'bg-1', projectId: 'g-1', name: '会议室', kind: 'background', originalName: 'room.webp', mimeType: 'image/webp', sizeBytes: 100, contentHash: 'a', url: '/room.webp', createdAt: '2026-09-23' },
  { id: 'portrait-1', projectId: 'g-1', name: '闻峥立绘', kind: 'portrait', originalName: 'wen.png', mimeType: 'image/png', sizeBytes: 100, contentHash: 'b', url: '/wen.png', createdAt: '2026-09-23' },
]
const characters: GameProjectCharacter[] = [{ id: 'wen', agentId: 1, displayName: '闻峥', role: '导演', characterType: 'director', visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], knownInformation: [], knowledgeRefs: [], relationIds: [], permissions: [] }]
const scene: GameProjectScene = { id: 's-1', name: '董事会议室', order: 0, backgroundAssetId: 'bg-1', soundEffectAssetIds: [], characters: [] }

function renderEditor(scenes: GameProjectScene[], onChange = vi.fn()) {
  render(<ConfigProvider theme={antdTheme}><App><GameScenesEditor assets={assets} scenes={scenes} characters={characters} onChange={onChange} onSave={vi.fn()} onUpload={vi.fn()} onReplaceAsset={vi.fn()} onDeleteAsset={vi.fn()}/></App></ConfigProvider>)
  return onChange
}

it('用策划能理解的文字展示场景工作区', () => {
  renderEditor([scene])
  expect(screen.getByDisplayValue('董事会议室')).toBeInTheDocument()
  expect(screen.getByText('画面与声音')).toBeInTheDocument()
  expect(screen.queryByText('从右侧选择背景图')).not.toBeInTheDocument()
  expect(screen.queryByText(/Schema|Manifest|Payload|内部ID/)).not.toBeInTheDocument()
})

it('可以新建场景并立即进入编辑', async () => {
  const onChange = renderEditor([])
  await userEvent.click(screen.getByRole('button', { name: '新建第一个场景' }))
  expect(onChange).toHaveBeenCalledWith([expect.objectContaining({ name: '新场景 1', order: 0, characters: [] })])
})

it('人物立绘位置用左中右选择，保存为稳定坐标', async () => {
  const onChange = renderEditor([scene])
  await userEvent.click(screen.getByLabelText('显示闻峥'))
  expect(onChange).toHaveBeenCalledWith([expect.objectContaining({ characters: [expect.objectContaining({ characterId: 'wen', x: 50, visible: true })] })])
})
