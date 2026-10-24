import { App, ConfigProvider } from 'antd'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { antdTheme } from '@/lib/antd-theme'
import GameObjectivesEditor from './GameObjectivesEditor'

const objectives = [{
  id: 'secure-evidence', name: '封存关键证据', description: '找到原始凭证并完成封存。', kind: 'main' as const,
  issuerCharacterId: 'legal', startsActive: false, dependsOn: [], conditionMode: 'all' as const,
  conditions: [{ target: 'evidence', operator: 'gte' as const, value: 60 }], deadlineTurns: 3,
}]

const metrics = [{ id: 'evidence', name: '证据完整度', type: 'integer' as const, initial: 20, minimum: 0, maximum: 100 }]
const characters = [{ id: 'legal', agentId: 2, displayName: '谢衡', role: '法律顾问', characterType: 'npc' as const, visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], knownInformation: [], knowledgeRefs: [], relationIds: [], permissions: [] }]

function renderEditor(props: Partial<React.ComponentProps<typeof GameObjectivesEditor>> = {}) {
  return render(<ConfigProvider theme={antdTheme}><App><GameObjectivesEditor objectives={objectives} metrics={metrics} characters={characters} onChange={vi.fn()} onSave={vi.fn()} {...props}/></App></ConfigProvider>)
}

it('用策划能看懂的语言展示任务规则', () => {
  renderEditor()
  expect(screen.getByText('封存关键证据')).toBeInTheDocument()
  expect(screen.getByText('主线任务')).toBeInTheDocument()
  expect(screen.getByText('由 谢衡 交付')).toBeInTheDocument()
  expect(screen.getByText('1 条完成条件')).toBeInTheDocument()
  expect(screen.getByText('3 回合内完成')).toBeInTheDocument()
  expect(screen.queryByText(/Schema|Payload|objectiveStatuses/)).not.toBeInTheDocument()
})

it('可以配置发布人物、前置任务、完成与失败条件', async () => {
  renderEditor({ objectives: [...objectives, { ...objectives[0], id: 'board-approval', name: '获得董事会授权', issuerCharacterId: undefined, startsActive: true }] })
  fireEvent.click(screen.getByRole('button', { name: '新建任务' }))
  expect(await screen.findByText('怎样算完成')).toBeInTheDocument()
  expect(screen.getByText('期限与失败')).toBeInTheDocument()
  expect(screen.getByText('玩家接受后记住的承诺')).toBeInTheDocument()
  expect(screen.getByText('开局直接显示')).toBeInTheDocument()
  expect(screen.getByText('先完成哪些任务')).toBeInTheDocument()
  fireEvent.click(screen.getByText('+添加完成条件'))
  await waitFor(() => expect(screen.getByRole('button', { name: '删除添加完成条件' })).toBeInTheDocument())
})
