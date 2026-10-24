import { expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { App, ConfigProvider } from 'antd'
import { antdTheme } from '@/lib/antd-theme'
import WorldRulesEditor from './WorldRulesEditor'

const preview = vi.fn()
vi.mock('@/queries/useGameProjects', () => ({
  usePreviewGameAction: () => ({ mutateAsync: preview, data: { allowed: true, reasons: ['当前条件允许执行这个行动'], stateDiff: [{ metricId: 'pressure', before: 40, after: 20 }] }, isPending: false, reset: vi.fn() }),
}))

const value = {
  metrics: [
    { id: 'wealth', name: '财富', description: '可支配资产', type: 'number' as const, initial: 100, minimum: 0, maximum: 200, displayUnit: '亿元' },
    { id: 'pressure', name: '监管压力', description: '监管关注程度', type: 'number' as const, initial: 40, minimum: 0, maximum: 100, displayUnit: '' },
  ],
  actions: [{
    id: 'disclose', name: '主动披露', description: '向监管机构公开情况', timeCost: 1,
    conditionMode: 'all' as const,
    conditions: [{ target: 'pressure', operator: 'gte' as const, value: 30 }],
    effects: [{ target: 'pressure', operation: 'decrement' as const, value: 20 }],
  }],
}

it('shows understandable state, action and trial sections', async () => {
  const user = userEvent.setup()
  preview.mockResolvedValue({})
  render(<ConfigProvider theme={antdTheme}><App><WorldRulesEditor projectId="speeding" value={value} onChange={vi.fn()}/></App></ConfigProvider>)
  expect(screen.getByText('世界状态')).toBeInTheDocument()
  expect(screen.getAllByText('财富').length).toBeGreaterThan(0)
  expect(screen.getAllByText('主动披露').length).toBeGreaterThan(0)
  await user.click(screen.getByRole('combobox', { name: '选择要试算的行动' }))
  await user.click(await screen.findByText('主动披露', { selector: '.ant-select-item-option-content' }))
  await user.click(screen.getByRole('button', { name: '开始试算' }))
  expect(await screen.findByText('可以执行这个行动')).toBeInTheDocument()
  expect(screen.getByText('40 → 20')).toBeInTheDocument()
  expect(screen.queryByText(/Schema|Manifest|Payload/)).not.toBeInTheDocument()
})

it('keeps rendering older saved actions when empty lists are omitted', () => {
  const legacyValue = {
    metrics: value.metrics,
    actions: [{ ...value.actions[0], conditions: undefined, effects: undefined }],
  }
  render(<ConfigProvider theme={antdTheme}><App><WorldRulesEditor projectId="speeding" value={legacyValue as unknown as typeof value} onChange={vi.fn()}/></App></ConfigProvider>)
  expect(screen.getByText('随时可执行')).toBeInTheDocument()
  expect(screen.getByText('0 项影响')).toBeInTheDocument()
})

it('lets a designer connect an action to the next story event', async () => {
  const user = userEvent.setup()
  const onChange = vi.fn()
  const events = [{
    id: 'crisis_resolution', name: '危机处置结果', category: 'route' as const,
    displayContent: '董事会给出处置决定', conditionMode: 'all' as const,
    conditions: [], participantIds: [], effects: [], advancesTime: false,
    timeAdvance: 0, priority: 10, once: true, cooldownTurns: 0,
    nextEventIds: [], endsGame: false,
  }]
  render(<ConfigProvider theme={antdTheme}><App><WorldRulesEditor projectId="speeding" value={value} events={events} onChange={onChange}/></App></ConfigProvider>)
  await user.click(screen.getByRole('button', { name: '编辑主动披露' }))
  expect(screen.getByText('执行后直接推动哪个故事事件')).toBeInTheDocument()
  await user.click(screen.getByRole('combobox', { name: '执行后直接推动哪个故事事件' }))
  await user.click(await screen.findByText('危机处置结果', { selector: '.ant-select-item-option-content' }))
  await user.click(screen.getByRole('button', { name: /完\s*成/ }))
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
    actions: [expect.objectContaining({ id: 'disclose', emits: 'crisis_resolution' })],
  }))
})
