import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { App, ConfigProvider } from 'antd'
import { beforeEach, expect, it, vi } from 'vitest'
import { antdTheme } from '@/lib/antd-theme'
import GameEventsEditor from './GameEventsEditor'

const mutateAsync = vi.fn()
vi.mock('@/queries/useGameProjects', () => ({
  usePreviewGameEvents: () => ({ mutateAsync, isPending: false }),
  useGameSessions: () => ({ data: [], isLoading: false, isError: false }),
  useGameSession: () => ({ data: undefined, isLoading: false, isError: false }),
  useCreateGameSession: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useAdvanceGameSession: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useTransitionGameSession: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

const event = {
  id: 'exposure',
  name: '隐藏债务被曝光',
  description: '验证玩家的危机处理',
  category: 'route' as const,
  displayContent: '一名记者拿到了集团隐藏债务的证据。',
  conditionMode: 'all' as const,
  conditions: [{ target: 'pressure', operator: 'gte' as const, value: 60 }],
  participantIds: ['cfo'],
  effects: [{ target: 'reputation', operation: 'decrement' as const, value: 15 }],
  advancesTime: true,
  timeAdvance: 1,
  priority: 80,
  once: true,
  cooldownTurns: 0,
  exclusiveGroup: 'crisis',
  nextEventIds: [],
  endsGame: false,
}

const value = {
  events: [event],
  metrics: [
    { id: 'pressure', name: '监管压力', type: 'number' as const, initial: 65, minimum: 0, maximum: 100 },
    { id: 'reputation', name: '公众信任', type: 'number' as const, initial: 70, minimum: 0, maximum: 100 },
  ],
  actions: [],
  characters: [{ id: 'cfo', agentId: 7, displayName: '莫云岑', role: '首席财务官', characterType: 'npc' as const, visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], knownInformation: [], knowledgeRefs: [], relationIds: [], permissions: [] }],
}

function renderEditor() {
  return render(<ConfigProvider theme={antdTheme}><App><GameEventsEditor projectId="speeding" value={value} onChange={vi.fn()} onSave={vi.fn()}/></App></ConfigProvider>)
}

beforeEach(() => mutateAsync.mockReset())

it('用策划看得懂的方式展示事件、触发和影响', () => {
  renderEditor()
  expect(screen.getByText('事件决定故事何时发生')).toBeInTheDocument()
  expect(screen.getByText('隐藏债务被曝光')).toBeInTheDocument()
  expect(screen.getByText('路线事件')).toBeInTheDocument()
  expect(screen.getByText('全部 1 个条件')).toBeInTheDocument()
  expect(screen.getByText('1 项数值影响')).toBeInTheDocument()
  expect(screen.queryByText(/Schema|Manifest|Payload|数据库ID/)).not.toBeInTheDocument()
})

it('新建事件时展示完整的故事编辑项', async () => {
  renderEditor()
  fireEvent.click(screen.getByRole('button', { name: '新建事件' }))
  expect(await screen.findByText('何时发生')).toBeInTheDocument()
  expect(screen.getByText('哪些人物会参与')).toBeInTheDocument()
  expect(screen.getByText('发生后，世界会怎么变')).toBeInTheDocument()
  expect(screen.getAllByText('仅发生一次').length).toBeGreaterThan(1)
  expect(screen.getByText('发生后结束游戏')).toBeInTheDocument()
})

it('打开试跑时要求输入回合与真实世界状态', () => {
  renderEditor()
  fireEvent.click(screen.getByRole('button', { name: '试跑隐藏债务被曝光' }))
  expect(screen.getByText('当前回合')).toBeInTheDocument()
  expect(screen.getByText('监管压力')).toBeInTheDocument()
  expect(screen.getByText('公众信任')).toBeInTheDocument()
  expect(screen.getByText(/\u7531\u771f\u5b9e\u6e38\u620f\u89c4\u5219\u5224\u65ad/)).toBeInTheDocument()
})

it('展示后端真实试跑返回的触发原因和数值变化', async () => {
  mutateAsync.mockResolvedValue({
    currentTurn: 3,
    selectedIds: ['exposure'],
    summary: '本回合选中 1 个事件',
    items: [{ eventId: 'exposure', name: '隐藏债务被曝光', category: 'route', eligible: true, selected: true, reasons: ['监管压力已达到触发线'], checks: [], before: { reputation: 70 }, after: { reputation: 55 }, stateDiff: [{ metricId: 'reputation', before: 70, after: 55 }], timeAdvance: 1, endsGame: false, priority: 80 }],
  })
  renderEditor()
  fireEvent.click(screen.getByRole('button', { name: '试跑隐藏债务被曝光' }))
  fireEvent.click(screen.getByRole('button', { name: '开始试跑' }))
  await waitFor(() => expect(mutateAsync).toHaveBeenCalledWith(expect.objectContaining({ currentTurn: 1, candidateEventIds: ['exposure'] })))
  expect(await screen.findByText('本回合选中 1 个事件')).toBeInTheDocument()
  expect(screen.getByText('监管压力已达到触发线')).toBeInTheDocument()
  expect(screen.getByText(/公众信任 70 → 55/)).toBeInTheDocument()
})
