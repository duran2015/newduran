import { describe, it, expect } from 'vitest'
import { getBreadcrumbs } from './nav'

describe('getBreadcrumbs', () => {
  it('returns just 首页 for the dashboard route', () => {
    expect(getBreadcrumbs('/dashboard')).toEqual([{ label: '首页' }])
    expect(getBreadcrumbs('/')).toEqual([{ label: '首页' }])
  })

  it('prefixes nav labels with 首页', () => {
    expect(getBreadcrumbs('/agents')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: 'Agent 管理' }
    ])
    expect(getBreadcrumbs('/mcps')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: 'MCP 接入' }
    ])
    expect(getBreadcrumbs('/groups')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '群组与频道' }
    ])
    expect(getBreadcrumbs('/governance')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '流程与决策' }
    ])
  })

  it('matches nested routes under a nav item', () => {
    expect(getBreadcrumbs('/knowledge/12/documents')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '知识库', path: '/knowledge' },
      { label: '详情' }
    ])
  })

  it('uses the current game-making step in the breadcrumb', () => {
    expect(getBreadcrumbs('/games/speeding-demo/events')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '游戏项目', path: '/games' },
      { label: '事件与推进' }
    ])
  })

  it('uses the knowledge base name when provided', () => {
    expect(getBreadcrumbs('/knowledge/12/documents', '开发专用知识库')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '知识库', path: '/knowledge' },
      { label: '开发专用知识库' }
    ])
  })

  it('returns setting section and page for settings routes', () => {
    expect(getBreadcrumbs('/settings/cli-tokens')).toEqual([
      { label: '首页', path: '/dashboard' },
      { label: '设置' },
      { label: 'CLI Tokens' }
    ])
  })

  it('falls back to 首页 for unknown routes', () => {
    expect(getBreadcrumbs('/nonexistent')).toEqual([{ label: '首页' }])
  })
})
