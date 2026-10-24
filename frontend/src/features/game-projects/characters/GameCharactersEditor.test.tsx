import { expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { App, ConfigProvider } from 'antd'
import { antdTheme } from '@/lib/antd-theme'
import GameCharactersEditor from './GameCharactersEditor'

vi.mock('@/queries/useAgents', () => ({
  useAgents: () => ({
    data: [{ id: 7, name: 'speeding-cfo', config: { title: { 'zh-CN': '莫云岑' } }, datasets: ['finance'] }],
    isLoading: false,
    isError: false,
  }),
}))

vi.mock('@/queries/useAgentRelations', () => ({
  useAgentRelations: () => ({ data: [], isLoading: false }),
}))

const character = {
  id: 'cfo',
  agentId: 7,
  displayName: '莫云岑',
  role: '首席财务官',
  responsibilities: '评估资金与资产处置风险',
  characterType: 'npc' as const,
  visibleMetricIds: ['wealth'],
  executableActionIds: ['freeze'],
  advisoryActionIds: ['disclose'],
  knownInformation: ['公司存在隐藏债务'],
  informationScope: '仅限这个人物',
  knowledgeRefs: ['finance'],
  relationIds: [],
  permissions: [],
}

it('explains that a game references an existing agent and shows game permissions', () => {
  render(<ConfigProvider theme={antdTheme}><App><GameCharactersEditor projectId="speeding" value={{
    characters: [character],
    metrics: [{ id: 'wealth', name: '财富', type: 'number', initial: 100 }],
    actions: [
      { id: 'freeze', name: '冻结交易', conditionMode: 'all', conditions: [], effects: [], timeCost: 0 },
      { id: 'disclose', name: '主动披露', conditionMode: 'all', conditions: [], effects: [], timeCost: 0 },
    ],
  }} onChange={vi.fn()}/></App></ConfigProvider>)

  expect(screen.getByText('游戏引用 Agent，不会复制 Agent')).toBeInTheDocument()
  expect(screen.getByText('莫云岑')).toBeInTheDocument()
  expect(screen.getByText('首席财务官')).toBeInTheDocument()
  expect(screen.getByText('可执行 1 个行动')).toBeInTheDocument()
  expect(screen.getByText('可建议 1 个行动')).toBeInTheDocument()
  expect(screen.queryByText(/Schema|JSON|内部ID/)).not.toBeInTheDocument()
})

it('warns when the referenced agent is no longer available', () => {
  render(<ConfigProvider theme={antdTheme}><App><GameCharactersEditor projectId="speeding" value={{
    characters: [{ ...character, agentId: 999 }], metrics: [], actions: [],
  }} onChange={vi.fn()}/></App></ConfigProvider>)

  expect(screen.getByText('Agent Hub 中已找不到这个 Agent，请重新选择后再试玩。')).toBeInTheDocument()
})
