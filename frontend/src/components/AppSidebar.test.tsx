import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import AppSidebar from './AppSidebar'
import { NAV_ITEMS } from '@/lib/nav'
// 结构性断言（i18n P3）：NAV_ITEMS.label 已 key 化，渲染输出为翻译值——
// 自反断言须同包 i18next.t 保持「数据→渲染」一致性语义。
import i18next from '@/i18n'

function renderSidebar(props: { collapsed?: boolean } = {}, route = '/dashboard') {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <AppSidebar collapsed={props.collapsed ?? false} />
    </MemoryRouter>
  )
}

describe('AppSidebar', () => {
  it('groups navigation by user intent and keeps every section visible', () => {
    renderSidebar()
    for (const label of ['工作台', 'Agent Hub', '协作中心', '游戏编辑器', '扩展与运营']) expect(screen.getByText(label)).toBeInTheDocument()
    for (const item of ['工作概览', 'Agent 管理', '人格库', '模型服务', '运行记录', '群组与频道', '游戏项目', '用量与告警']) expect(screen.getByText(item)).toBeInTheDocument()
    expect(screen.queryByText('研发验收')).not.toBeInTheDocument()
  })

  it('marks the nav item matching the current route as active', () => {
    renderSidebar({}, '/agents')
    const active = screen.getByRole('button', { name: /Agent 管理/ })
    expect(active).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('Agent Hub')).toBeInTheDocument()
  })

  it('shows the six game-making steps as second-level navigation inside a game', () => {
    renderSidebar({}, '/games/speeding-demo/events')
    const steps = screen.getByLabelText('当前游戏制作步骤')
    for (const label of ['游戏概览', '规则与世界状态', 'Agent / NPC', '事件与推进', '场景与视听', '试玩与发布']) {
      expect(within(steps).getByRole('button', { name: new RegExp(label) })).toBeInTheDocument()
    }
    expect(within(steps).getByRole('button', { name: /事件与推进/ })).toHaveAttribute('aria-current', 'step')
  })

  it('hides labels and shows only icons when collapsed', () => {
    renderSidebar({ collapsed: true })
    for (const item of NAV_ITEMS) {
      expect(screen.queryByText(i18next.t(item.label))).not.toBeInTheDocument()
      // icon-only buttons still expose an accessible name
      expect(screen.getByRole('button', { name: i18next.t(item.label) })).toBeInTheDocument()
    }
  })
})
