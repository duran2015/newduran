import { App, ConfigProvider } from 'antd'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { antdTheme } from '@/lib/antd-theme'
import GamePlayerPage from './GamePlayerPage'

const navigate = vi.fn()
const confirm = vi.fn()
const interpret = vi.fn()
const reject = vi.fn()
const create = vi.fn()
const reset = vi.fn()
const acceptObjective = vi.fn()
const rejectObjective = vi.fn()
let routeParams: { projectId: string; sessionId?: string } = { projectId: 'game-1', sessionId: 'session-1' }
let sessionData: object | undefined

vi.mock('react-router', async () => ({
  ...(await vi.importActual<object>('react-router')),
  useNavigate: () => navigate,
  useParams: () => routeParams,
}))

vi.mock('@/queries/useGameProjects', () => ({
  useGameProject: () => ({ data: { project: { id: 'game-1', name: '危机之夜', description: '在倒计时结束前解决董事会危机。', revision: 3 }, draft: { metrics: [{ id: 'trust', name: '信任', type: 'integer', initial: 50 }], npcs: [], actions: [], events: [], scenes: [] } }, isLoading: false, isError: false }),
  useGameProjectAssets: () => ({ data: [] }),
}))

vi.mock('@/queries/usePlayerGames', () => ({
  usePublishedGame: () => ({ data: undefined, isLoading: false, isError: false }),
  usePlayerGameSession: () => ({ data: sessionData, isLoading: false, isError: false }),
  useCreatePlayerGameSession: () => ({ mutateAsync: create, isPending: false, isError: false }),
  useResetPlayerGameSession: () => ({ mutateAsync: reset, isPending: false }),
  useInterpretPlayerAction: () => ({ mutateAsync: interpret, isPending: false }),
  useConfirmPlayerAction: () => ({ mutateAsync: confirm, isPending: false }),
  useRejectPlayerAction: () => ({ mutateAsync: reject, isPending: false }),
  useAcceptPlayerObjective: () => ({ mutateAsync: acceptObjective, isPending: false }),
  useRejectPlayerObjective: () => ({ mutateAsync: rejectObjective, isPending: false }),
}))

beforeEach(() => { vi.clearAllMocks(); routeParams = { projectId: 'game-1', sessionId: 'session-1' }; sessionData = { id: 'session-1', projectId: 'game-1', source: 'draft', projectRevision: 3, status: 'running', currentTurn: 1, state: { trust: 45 }, scenes: [], timeline: [{ label: '游戏开始', content: '董事会即将召开。', occurredAt: '2026-09-23T00:00:00Z' }], pendingPlan: { id: 'plan-1', summary: '先核对财务缺口', warnings: [], needsConfirmation: true, effects: [{ metricId: 'trust', before: 45, after: 50 }] } }; confirm.mockResolvedValue({}); reject.mockResolvedValue({}); interpret.mockResolvedValue({}); acceptObjective.mockResolvedValue({}); rejectObjective.mockResolvedValue({}); create.mockResolvedValue({ id: 'new-session' }); reset.mockResolvedValue({ id: 'reset-session', projectId: 'game-1' }) })

it('在改变世界状态前要求玩家确认系统整理出的行动', () => {
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByText('系统整理出的行动')).toBeInTheDocument()
  expect(screen.getByText('先核对财务缺口')).toBeInTheDocument()
  expect(screen.getByText('信任：45 → 50')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: /确认执行/ }))
  expect(confirm).toHaveBeenCalledWith(expect.objectContaining({ projectId: 'game-1', sessionId: 'session-1', planId: 'plan-1' }))
})

it('允许玩家拒绝错误理解，且不会在页面伪造人物回复', () => {
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.queryByText(/NPC 已回复/)).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: /重新说/ }))
  expect(reject).toHaveBeenCalledWith(expect.objectContaining({ planId: 'plan-1' }))
})

it('可以新建一局并明确从头开始', async () => {
  routeParams = { projectId: 'game-1' }
  sessionData = undefined
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  fireEvent.click(screen.getByRole('button', { name: /新建试玩/ }))
  expect(create).toHaveBeenCalledWith(expect.objectContaining({ projectId: 'game-1', startAt: { kind: 'beginning' } }))
})

it('使用事件指定场景，不随回合自动跳场景，并显示结局及复盘', () => {
  sessionData = { id: 'session-1', projectId: 'game-1', source: 'draft', status: 'completed', currentTurn: 99, currentSceneId: 'board', state: { trust: 30 }, scenes: [{ id: 'board', name: '董事会议室', characters: [] }, { id: 'street', name: '街道', characters: [] }], ending: { title: '交易暂停', summary: '董事会决定先完成调查。' }, timeline: [{ label: '调查完成', content: '查清了资金来源。', occurredAt: '2026-09-23T00:00:00Z', stateDiff: [{ metricId: 'trust', before: 50, after: 30 }] }] }
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByRole('heading', { name: '董事会议室' })).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: '街道' })).not.toBeInTheDocument()
  expect(screen.getByText('交易暂停')).toBeInTheDocument()
  expect(screen.getByLabelText('本局复盘')).toHaveTextContent('信任：50 → 30')
  expect(screen.queryByLabelText('告诉游戏你想做什么')).not.toBeInTheDocument()
})

it('需要澄清的理解不能执行', () => {
  sessionData = { ...(sessionData as object), pendingPlan: { id: 'clarify', summary: '请说明你想找谁调查。', needsConfirmation: false, warnings: [] } }
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByText('请说明你想找谁调查。')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /确认执行/ })).not.toBeInTheDocument()
})

it('普通对话直接进入人物回应而不要求二次确认', async () => {
  sessionData = { ...(sessionData as object), pendingPlan: null, scenes: [{ id: 'room', name: '会议室', characters: [{ characterId: 'finance', name: '莫云岑', x: 50, y: 0, scale: 1, layer: 1, visible: true }] }] }
  interpret.mockResolvedValue({ ...(sessionData as object), pendingPlan: { id: 'talk-1', actionId: '__freeform_interaction__', summary: '尝试与莫云岑交谈', warnings: [], needsConfirmation: true, effects: [] } })
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  fireEvent.change(screen.getByLabelText('对莫云岑说'), { target: { value: '我想和你谈谈' } })
  fireEvent.click(screen.getByRole('button', { name: '发送' }))
  await waitFor(() => expect(confirm).toHaveBeenCalledWith(expect.objectContaining({ planId: 'talk-1' })))
  expect(interpret).toHaveBeenCalledWith(expect.objectContaining({ targetCharacterId: 'finance' }))
})

it('人物临时没有回应时可以原地重试上一句话', async () => {
  sessionData = { ...(sessionData as object), pendingPlan: null, scenes: [{ id: 'room', name: '会议室', characters: [{ characterId: 'finance', name: '莫云岑', visible: true }] }], timeline: [
    { id: 'player-1', label: '玩家', kind: 'player', speaker: '你', content: '把你知道的资金路径告诉我。', targetCharacterId: 'finance', occurredAt: '2026-09-23T00:00:00Z' },
    { id: 'npc-1', label: '莫云岑', kind: 'npc', speaker: '莫云岑', content: '莫云岑回应超时，故事可以继续。', characterId: 'finance', occurredAt: '2026-09-23T00:00:01Z' },
  ] }
  interpret.mockResolvedValue({ ...(sessionData as object), pendingPlan: { id: 'retry-1', actionId: '__freeform_interaction__', summary: '再次询问莫云岑', warnings: [], needsConfirmation: true, effects: [] } })
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByText('莫云岑刚才没有成功回应')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '重新询问这句话' }))
  await waitFor(() => expect(interpret).toHaveBeenCalledWith(expect.objectContaining({ text: '把你知道的资金路径告诉我。', targetCharacterId: 'finance' })))
  await waitFor(() => expect(confirm).toHaveBeenCalledWith(expect.objectContaining({ planId: 'retry-1' })))
})

it('清楚展示进行中、完成、失败和未解锁任务', () => {
  sessionData = { ...(sessionData as object), pendingPlan: null, objectives: [
    { id: 'a', name: '封存证据', description: '保住所有原始凭证。', kind: 'main', status: 'active', issuer: '谢衡' },
    { id: 'b', name: '获得授权', kind: 'main', status: 'completed', issuer: '莫云岑' },
    { id: 'c', name: '控制舆情', kind: 'optional', status: 'failed' },
    { id: 'd', name: '面对监管', kind: 'main', status: 'locked' },
  ] }
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByText('进行中')).toBeInTheDocument()
  expect(screen.getByText('已完成')).toBeInTheDocument()
  expect(screen.getByText('已失败')).toBeInTheDocument()
  expect(screen.getByText('尚未解锁')).toBeInTheDocument()
  expect(screen.getAllByText('来自 谢衡').length).toBeGreaterThan(0)
  expect(screen.getByText('保住所有原始凭证。')).toBeInTheDocument()
})

it('将人物提出的任务、承诺和风险交给玩家明确决定', async () => {
  sessionData = { ...(sessionData as object), pendingPlan: null, objectives: [{ id: 'report', name: '准备监管说明', description: '补齐证据后向监管方提交说明。', kind: 'optional', status: 'offered', issuer: '谢衡', promiseText: '我会在董事会前交出完整说明。' }] }
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  expect(screen.getByText('谢衡向你提出一项任务')).toBeInTheDocument()
  expect(screen.getByText(/接受即代表承诺/).parentElement).toHaveTextContent('董事会前交出完整说明')
  expect(screen.getByText(/拒绝会留下记录，但不算任务失败/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '接受任务' }))
  await waitFor(() => expect(acceptObjective).toHaveBeenCalledWith(expect.objectContaining({ objectiveId: 'report', requestKey: expect.any(String) })))
})

it('拒绝任务使用独立决定接口，接口失败时可原地重试', async () => {
  rejectObjective.mockRejectedValueOnce(new Error('网络中断'))
  sessionData = { ...(sessionData as object), pendingPlan: null, objectives: [{ id: 'report', name: '准备监管说明', kind: 'optional', status: 'offered', issuer: '谢衡' }] }
  render(<ConfigProvider theme={antdTheme}><App><GamePlayerPage/></App></ConfigProvider>)
  fireEvent.click(screen.getByRole('button', { name: '拒绝这次请求' }))
  await waitFor(() => expect(rejectObjective).toHaveBeenCalledWith(expect.objectContaining({ objectiveId: 'report', requestKey: expect.any(String) })))
  expect(await screen.findByText('这次选择还没有生效')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '拒绝这次请求' })).toBeEnabled()
})
