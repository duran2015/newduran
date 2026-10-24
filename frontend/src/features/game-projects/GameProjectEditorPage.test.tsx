import { expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { App, ConfigProvider } from 'antd'
import GameProjectEditorPage from './GameProjectEditorPage'
import { antdTheme } from '@/lib/antd-theme'

const update = vi.fn()
let currentStep: string | undefined
const projectDetail = { project: { id: 'speeding-demo', name: 'Speeding 小型故事', description: '玩家处置财富并面对社会反应', genre: '财富策略', playGoal: '守住控制权', status: 'draft', revision: 2 }, draft: { metrics: [], actions: [], endings: [], npcs: [], events: [], scenes: [] }, releases: [] }

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return { ...actual, useParams: () => ({ projectId: 'speeding-demo', step: currentStep }), useNavigate: () => vi.fn() }
})

vi.mock('@/queries/useGameProjects', () => ({
  useGameProject: () => ({ data: projectDetail }),
  useUpdateGameProject: () => ({ mutateAsync: update, isPending: false }),
  useGameProjectAssets: () => ({ data: [], isError: false }),
  useUploadGameProjectAsset: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useReplaceGameProjectAsset: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteGameProjectAsset: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock('@/queries/useAgents', () => ({
  useAgents: () => ({ data: [], isLoading: false, isError: false }),
}))

vi.mock('@/queries/useAgentRelations', () => ({
  useAgentRelations: () => ({ data: [], isLoading: false, isError: false }),
}))

it('keeps only the selected step content in the editor body', async () => {
  currentStep = 'characters'
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<ConfigProvider theme={antdTheme}><App><QueryClientProvider client={client}><MemoryRouter><GameProjectEditorPage /></MemoryRouter></QueryClientProvider></App></ConfigProvider>)
  expect(await screen.findByText('Speeding 小型故事')).toBeInTheDocument()
  expect(screen.getByText('还没有为这款游戏添加人物')).toBeInTheDocument()
  expect(screen.getByText(/游戏引用 Agent，不会复制 Agent/)).toBeInTheDocument()
  expect(screen.queryByRole('navigation', { name: '游戏制作步骤' })).not.toBeInTheDocument()
  expect(screen.queryByText('Manifest')).not.toBeInTheDocument()
})

it('edits and saves the game overview', async () => {
  currentStep = undefined
  update.mockResolvedValue({})
  const user = userEvent.setup()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<ConfigProvider theme={antdTheme}><App><QueryClientProvider client={client}><MemoryRouter><GameProjectEditorPage /></MemoryRouter></QueryClientProvider></App></ConfigProvider>)
  const name = await screen.findByLabelText('游戏名称')
  await user.clear(name)
  await user.type(name, 'Speeding 正式小样')
  await user.click(screen.getByRole('button', { name: '保存游戏资料' }))
  expect(update).toHaveBeenCalledWith(expect.objectContaining({ revision: 2, name: 'Speeding 正式小样', playGoal: '守住控制权' }))
})
