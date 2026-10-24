import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { App, ConfigProvider } from 'antd'
import { beforeEach, expect, it, vi } from 'vitest'
import { antdTheme } from '@/lib/antd-theme'
import GameSessionWorkspace from './GameSessionWorkspace'

const create = vi.fn()
const advance = vi.fn()
const transition = vi.fn()
let sessionData: any
let sessionsData: any[] = []

vi.mock('@/queries/useGameProjects', () => ({
  useGameSessions: () => ({ data: sessionsData, isLoading: false, isError: false, refetch: vi.fn() }),
  useGameSession: () => ({ data: sessionData, isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateGameSession: () => ({ mutateAsync: create, isPending: false }),
  useAdvanceGameSession: () => ({ mutateAsync: advance, isPending: false }),
  useTransitionGameSession: () => ({ mutateAsync: transition, isPending: false }),
}))

const metrics = [
  { id: 'wealth', name: '可用资金', type: 'number' as const, initial: 80, displayUnit: '亿元' },
  { id: 'pressure', name: '监管压力', type: 'number' as const, initial: 20 },
]
const characters = [
  { id: 'director', agentId: 1, displayName: '闻峥', role: '危机总指挥', characterType: 'director' as const, visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], knownInformation: [], knowledgeRefs: [], relationIds: [], permissions: [] },
  { id: 'cfo', agentId: 2, displayName: '莫云岑', role: '首席财务官', characterType: 'npc' as const, visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], knownInformation: [], knowledgeRefs: [], relationIds: [], permissions: [] },
]

function renderWorkspace() {
  return render(<ConfigProvider theme={antdTheme}><App><GameSessionWorkspace projectId="speeding" metrics={metrics} characters={characters}/></App></ConfigProvider>)
}

beforeEach(() => {
  create.mockReset()
  advance.mockReset()
  transition.mockReset()
  sessionData = undefined
  sessionsData = []
})

it('在没有记录时引导策划开始第一局试玩', () => {
  renderWorkspace()
  expect(screen.getByText('完整试玩')).toBeInTheDocument()
  expect(screen.getByText('还没有完整试玩记录')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '开始第一局试玩' })).toBeInTheDocument()
  expect(screen.queryByText(/Schema|payload|runtime/)).not.toBeInTheDocument()
})

it('展示当前回合、世界状态、最新事件和完整过程', async () => {
  const session = {
    id: 'session-1', projectId: 'speeding', projectRevision: 3, status: 'running', directorCharacterId: 'director', currentTurn: 2,
    state: { wealth: 65, pressure: 42 }, createdAt: '2026-09-23T02:00:00Z', updatedAt: '2026-09-23T02:05:00Z',
    latestEvent: { name: '债务文件流出', displayContent: '记者拿到了关键文件。', participants: [{ characterId: 'cfo', displayName: '莫云岑' }], stateDiff: [{ metricId: 'pressure', before: 20, after: 42 }] },
    timeline: [{ id: 'log-1', kind: 'event_triggered', name: '债务文件流出', output: { summary: '事件发生并完成数值结算。' }, occurredAt: '2026-09-23T02:04:00Z' }],
  }
  sessionData = session
  sessionsData = [session]
  advance.mockResolvedValue(session)
  const user = userEvent.setup()
  renderWorkspace()
  expect(await screen.findByText('第 2 回合')).toBeInTheDocument()
  expect(screen.getByText('65 亿元')).toBeInTheDocument()
  expect(screen.getAllByText('债务文件流出')).toHaveLength(2)
  expect(screen.getByText('记者拿到了关键文件。')).toBeInTheDocument()
  expect(screen.getByText('监管压力：20 → 42')).toBeInTheDocument()
  expect(screen.getByText('莫云岑')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '推进下一回合' }))
  expect(advance).toHaveBeenCalledWith(expect.objectContaining({ projectId: 'speeding', sessionId: 'session-1', actorCharacterId: 'director' }))
})

it('暂停状态只提供继续或结束，不允许继续推进', () => {
  const session = { id: 'session-2', projectId: 'speeding', projectRevision: 3, status: 'paused', directorCharacterId: 'director', currentTurn: 4, state: {}, timeline: [], createdAt: '2026-09-23T02:00:00Z', updatedAt: '2026-09-23T02:05:00Z' }
  sessionData = session
  sessionsData = [session]
  renderWorkspace()
  expect(screen.getByRole('button', { name: '继续试玩' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '结束' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '推进下一回合' })).not.toBeInTheDocument()
})
