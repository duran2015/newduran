import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ConfigProvider } from 'antd'
import { antdTheme } from '@/lib/antd-theme'
import GameMachineLabPage from './GameMachineLabPage'

const setEnabled = vi.fn()
const runTest = vi.fn()

const packages = [
  { id: 'counter-v1', key: 'io.zerone.counter', version: '1.0.0', displayName: '计数器', description: '验证最小动作和状态变化', status: 'enabled', enabled: true, contentHash: 'abc', registry: { metricCount: 1, actionCount: 1, eventCount: 1, endingCount: 0 } },
  { id: 'restaurant-v1', key: 'io.zerone.restaurant', version: '1.0.0', displayName: '餐厅经营', description: '验证不同领域游戏包互不影响', status: 'disabled', enabled: false, contentHash: 'def', registry: { metricCount: 3, actionCount: 2, eventCount: 2, endingCount: 1 } },
]

const detail = {
  package: packages[0],
  manifest: {
    schemaVersion: 'game.zerone.io/v1',
    package: { key: 'io.zerone.counter', version: '1.0.0', displayName: '计数器', description: '最小测试包' },
    registry: {
      metrics: [{ id: 'count', name: '当前计数', type: 'number', initial: 0 }],
      actions: [{ id: 'increment', name: '增加计数', description: '将输入数字加到计数上', effects: [], emits: 'counter.incremented', exampleInput: { amount: 1 } }],
      events: [{ id: 'counter.incremented', name: '计数已增加' }],
      endings: [],
    },
    acceptanceScenarios: [{ id: 'counter-basic', name: '连续增加两次', commands: [{ actionId: 'increment', input: { amount: 1 } }, { actionId: 'increment', input: { amount: 2 } }] }],
  },
}

vi.mock('@/queries/useGamePackages', () => ({
  useGamePackages: () => ({ data: { items: packages, total: 2 }, isLoading: false, isError: false }),
  useGamePackage: (id?: string) => ({ data: id ? detail : undefined, isLoading: false, isError: false }),
  useSetGamePackageEnabled: () => ({ mutateAsync: setEnabled, isPending: false }),
  useInstallGameAcceptanceFixtures: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useGamePackageTestRun: () => ({
    mutateAsync: runTest,
    isPending: false,
    reset: vi.fn(),
    data: {
      packageId: 'counter-v1', packageVersion: '1.0.0', initialState: { count: 0 }, finalState: { count: 3 }, passed: true,
      steps: [{ sequence: 1, command: { actionId: 'increment', input: { amount: 1 } }, event: { id: 'counter.incremented', sequence: 1, actionId: 'increment' }, stateDiff: [{ metricId: 'count', before: 0, after: 1 }], state: { count: 1 } }],
    },
  }),
}))

function renderPage() {
  return render(<ConfigProvider theme={antdTheme}><GameMachineLabPage /></ConfigProvider>)
}

describe('GameMachineLabPage', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('shows installed packages and product-readable registry content', async () => {
    renderPage()
    expect(screen.getByRole('heading', { name: '游戏机器实验室' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '为什么有“计数器”和“餐厅”？' })).toBeInTheDocument()
    expect(screen.getByText(/Speeding 的财富、人物和事件规则/)).toBeInTheDocument()
    expect(screen.getAllByText('计数器')).toHaveLength(2)
    expect(screen.getByText('餐厅经营')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('这个包为机器提供了什么')).toBeInTheDocument())
    expect(screen.getByText('当前计数')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('tab', { name: '可执行操作 1' }))
    expect(screen.getAllByText('增加计数').length).toBeGreaterThan(0)
  })

  it('runs the package acceptance scenario and shows command event state evidence', async () => {
    renderPage()
    const runButton = await screen.findByRole('button', { name: /开始验证：连续增加两次/ })
    fireEvent.click(runButton)
    expect(runTest).toHaveBeenCalledWith({ id: 'counter-v1', commands: detail.manifest.acceptanceScenarios[0].commands, initialState: undefined })
    expect(screen.getByText('验证通过：机器完成了 1 个步骤，结果与规则一致')).toBeInTheDocument()
    expect(screen.getByText('计数已增加')).toBeInTheDocument()
    expect(screen.getByText('当前计数：0 → 1')).toBeInTheDocument()
    expect(screen.queryByText('counter.incremented')).not.toBeInTheDocument()
  })

  it('lets a PM disable one package independently', async () => {
    setEnabled.mockResolvedValue(undefined)
    renderPage()
    const toggle = await screen.findByRole('switch', { name: '启用游戏包' })
    fireEvent.click(toggle)
    await waitFor(() => { expect(setEnabled).toHaveBeenCalledWith({ id: 'counter-v1', enabled: false }) })
  })
})
