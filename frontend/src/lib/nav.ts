import type { Icon } from '@phosphor-icons/react'
// 纯函数上下文：getBreadcrumbs 直调 i18next 输出用户面文案
import i18next from '@/i18n'
import {
  GaugeIcon,
  RobotIcon,
  WrenchIcon,
  PlugsConnectedIcon,
  SparkleIcon,
  CubeIcon,
  FilmSlateIcon,
  BooksIcon,
  ChatsIcon,
  ShareNetworkIcon,
  FingerprintIcon,
  PackageIcon,
  PulseIcon,
  ShieldCheckIcon,
  UsersThreeIcon,
  GitBranchIcon,
  SquaresFourIcon,
  ChartPieIcon,
  GameControllerIcon,
  FlaskIcon,
  FlagCheckeredIcon
} from '@phosphor-icons/react'

export const GAME_EDITOR_STEPS = [
  { key: 'overview', label: '游戏概览', hint: '名称、题材和玩家目标', icon: SparkleIcon },
  { key: 'world', label: '规则与世界状态', hint: '核心数值和变化规则', icon: GaugeIcon },
  { key: 'characters', label: 'Agent / NPC', hint: '人物、身份、权限与情报', icon: RobotIcon },
  { key: 'events', label: '事件与推进', hint: '触发条件、影响和时间推进', icon: BooksIcon },
  { key: 'scenes', label: '场景与视听', hint: '场景、立绘、音乐和音效', icon: FilmSlateIcon },
  { key: 'release', label: '试玩与发布', hint: '试玩检查并生成发布版本', icon: FlagCheckeredIcon },
] as const

export interface NavItem {
  id: string
  label: string
  path: string
  icon: Icon
}

export interface NavGroup {
  id: string
  label: string
  items: readonly NavItem[]
}

/**
 * Static navigation items. Not reactive — derived from constants, not a store.
 */
export const NAV_ITEMS: readonly NavItem[] = [
  { id: 'dashboard', label: 'nav.dashboard', path: '/dashboard', icon: GaugeIcon },
  { id: 'games', label: 'nav.games', path: '/games', icon: GameControllerIcon },
  { id: 'runs', label: 'nav.runs', path: '/runs', icon: PulseIcon },
  { id: 'agents', label: 'nav.agents', path: '/agents', icon: RobotIcon },
  { id: 'personalities', label: 'nav.personalities', path: '/personalities', icon: FingerprintIcon },
  { id: 'relations', label: 'nav.relations', path: '/relations', icon: ShareNetworkIcon },
  { id: 'groups', label: 'nav.groups', path: '/groups', icon: UsersThreeIcon },
  { id: 'governance', label: 'nav.governance', path: '/governance', icon: GitBranchIcon },
  { id: 'collaboration-audit', label: 'nav.collaborationAudit', path: '/collaboration-audit', icon: ShieldCheckIcon },
  { id: 'extensions', label: 'nav.extensions', path: '/extensions', icon: PackageIcon },
  { id: 'game-lab', label: 'nav.gameLab', path: '/game-lab', icon: FlaskIcon },
  { id: 'templates', label: 'nav.templates', path: '/templates', icon: SquaresFourIcon },
  { id: 'usage', label: 'nav.usage', path: '/usage', icon: ChartPieIcon },
  { id: 'tools', label: 'nav.tools', path: '/tools', icon: WrenchIcon },
  { id: 'mcps', label: 'nav.mcps', path: '/mcps', icon: PlugsConnectedIcon },
  { id: 'skills', label: 'nav.skills', path: '/skills', icon: SparkleIcon },
  { id: 'providers', label: 'nav.providers', path: '/providers', icon: CubeIcon },
  { id: 'knowledge', label: 'nav.knowledge', path: '/knowledge', icon: BooksIcon },
  { id: 'scenes', label: 'nav.scenes', path: '/scenes', icon: FilmSlateIcon },
  { id: 'chat', label: 'nav.chat', path: '/chat', icon: ChatsIcon }
] as const

const itemsByID = new Map(NAV_ITEMS.map((item) => [item.id, item]))

function navItems(...ids: string[]): NavItem[] {
  return ids.map((id) => itemsByID.get(id)).filter((item): item is NavItem => Boolean(item))
}

/**
 * Product-facing information architecture. Groups follow the user's working
 * intent instead of implementation layers; every group stays visible in the
 * sidebar so routes do not appear to move when the current page changes.
 */
export const NAV_GROUPS: readonly NavGroup[] = [
  { id: 'workspace', label: '工作台', items: navItems('dashboard') },
  { id: 'agent-hub', label: 'Agent Hub', items: navItems('agents', 'personalities', 'knowledge', 'providers', 'tools', 'mcps', 'skills', 'scenes', 'chat') },
  { id: 'collaboration', label: '协作中心', items: navItems('runs', 'relations', 'groups', 'governance', 'collaboration-audit') },
  { id: 'game-editor', label: '游戏编辑器', items: navItems('games') },
  { id: 'platform', label: '扩展与运营', items: navItems('templates', 'extensions', 'usage') },
] as const

export function getNavGroup(pathname: string): NavGroup {
  return NAV_GROUPS.find((group) => group.items.some((item) => pathname.startsWith(item.path))) ?? NAV_GROUPS[0]
}

// settings 子页面包屑标签。值统一过 i18next.t()（cli-tokens 为英文品牌词，
// 无对应资源 key，t() 按 missing-key 透传原样输出）。
const SETTINGS_LABELS: Record<string, string> = {
  'cli-tokens': 'CLI Tokens',
  aigc: 'nav.settingsLabels.aigc',
  users: 'nav.settingsLabels.users',
  'audit-logs': 'nav.settingsLabels.auditLogs'
}

export interface BreadcrumbItem {
  label: string
  /** Route to navigate to when clicked. Absent = current page / non-navigable. */
  path?: string
}

/**
 * Breadcrumb items for a pathname, e.g. '/settings/aigc' ->
 * [{label:'首页',path:'/dashboard'}, {label:'设置'}, {label:'AIGC 标识配置'}].
 * The last item is the current page and never has a path.
 * `knowledgeName` replaces the generic '详情' segment on knowledge detail routes.
 */
export function getBreadcrumbs(pathname: string, knowledgeName?: string): BreadcrumbItem[] {
  // 纯函数直调 i18next（非组件上下文）：输出即用户面文案——
  // 消费方（AppHeader breadcrumb）无需再包 t()，nav.test 断言中文零改动。
  const home: BreadcrumbItem = { label: i18next.t('nav.home'), path: '/dashboard' }
  const segments = pathname.split('/').filter(Boolean)
  const [first, second, third] = segments
  if (!first || first === 'dashboard') return [{ label: i18next.t('nav.home') }]

  if (first === 'settings') {
    const page = second ? SETTINGS_LABELS[second] : undefined
    return page
      ? [home, { label: i18next.t('nav.settings') }, { label: i18next.t(page) }]
      : [home, { label: i18next.t('nav.settings') }]
  }

  const item = NAV_ITEMS.find((i) => i.path === `/${first}`)
  if (!item) return [{ label: i18next.t('nav.home') }]
  if (!second) return [home, { label: i18next.t(item.label) }]
  if (first === 'games') {
    const currentStep = GAME_EDITOR_STEPS.find((step) => step.key === third) ?? GAME_EDITOR_STEPS[0]
    return [home, { label: i18next.t(item.label), path: item.path }, { label: currentStep.label }]
  }
  return [home, { label: i18next.t(item.label), path: item.path }, { label: knowledgeName ?? i18next.t('nav.detail') }]
}
