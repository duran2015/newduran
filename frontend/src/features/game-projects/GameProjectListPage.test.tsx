import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { ConfigProvider } from 'antd'
import GameProjectListPage from './GameProjectListPage'
import { antdTheme } from '@/lib/antd-theme'

const navigate = vi.fn()
const create = vi.fn()
vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return { ...actual, useNavigate: () => navigate }
})

vi.mock('@/queries/useGameProjects', () => ({
  useGameProjects: () => ({ data: [{ project: { id: 'speeding-demo', name: 'Speeding 小型故事', description: '玩家处置财富并面对社会反应', genre: '策略叙事', playGoal: '守住控制权', status: 'draft' }, draft: { metrics: [], actions: [], endings: [], npcs: [], events: [], scenes: [] }, releases: [] }] }),
  useCreateGameProject: () => ({ mutateAsync: create, isPending: false }),
}))

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<ConfigProvider theme={antdTheme}><QueryClientProvider client={client}><MemoryRouter><GameProjectListPage /></MemoryRouter></QueryClientProvider></ConfigProvider>)
}

describe('GameProjectListPage', () => {
  beforeEach(() => { navigate.mockReset(); create.mockReset(); create.mockResolvedValue({ project: { id: 'new-game' } }) })

  it('explains where a game starts and keeps packages out of the creation flow', async () => {
    renderPage()
    expect(await screen.findByText('Speeding 小型故事')).toBeInTheDocument()
    expect(screen.getByText('先写清题材、玩家要做什么，以及怎样算成功。')).toBeInTheDocument()
    expect(screen.queryByText('上传游戏包')).not.toBeInTheDocument()
    expect(screen.queryByText('Schema')).not.toBeInTheDocument()
  })

  it('creates a project from product-facing questions', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole('button', { name: '新建游戏' }))
    await user.type(screen.getByPlaceholderText('例如：Speeding'), '首富危机')
    await user.type(screen.getByPlaceholderText('玩家将面对什么局面？'), '处置财富并应对社会反应')
    await user.type(screen.getByPlaceholderText('例如：财富与权力策略叙事'), '策略叙事')
    await user.type(screen.getByPlaceholderText('玩家最终需要完成什么？'), '守住控制权')
    await user.click(screen.getByRole('button', { name: '创建并开始制作' }))
    expect(create).toHaveBeenCalledWith(expect.objectContaining({ name: '首富危机', playGoal: '守住控制权' }))
    expect(navigate).toHaveBeenCalledWith('/games/new-game')
  })
})
