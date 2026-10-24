import { Tooltip } from 'antd'
import { useNavigate, useLocation } from 'react-router'
import { createStyles } from 'antd-style'
import { useTranslation } from 'react-i18next'
import { GAME_EDITOR_STEPS, NAV_GROUPS, NAV_ITEMS } from '@/lib/nav'
import { tokens as tk } from '@/styles/tokens'
import BrandMark from '@/components/BrandMark'
import ExtensionSidebarSection from '@/components/extensions/ExtensionSidebarSection'

const useStyles = createStyles(({ css }) => ({
  sidebar: css`
    width: 224px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    background: var(--sidebar);
    position: relative;
    z-index: 110;
    border-right: 1px solid color-mix(in srgb, var(--sidebar-border) 72%, transparent);
    box-shadow: 8px 0 18px -14px color-mix(in srgb, var(--foreground) 24%, transparent);
    transition: width 0.2s ease;
    overflow: hidden;
    @media (max-width: 768px) {
      display: none;
    }
  `,
  sidebarCollapsed: css`
    width: 64px;
  `,
  logo: css`
    display: flex;
    align-items: center;
    gap: 10px;
    height: 52px;
    padding: 0 20px;
    color: var(--sidebar-primary);
    flex-shrink: 0;
  `,
  logoCollapsed: css`
    padding: 0;
    justify-content: center;
  `,
  logoText: css`
    font-size: ${tk.textXl};
    font-weight: 700;
    letter-spacing: -0.02em;
    line-height: 1;
    white-space: nowrap;
  `,
  nav: css`
    flex: 1;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    overflow-y: auto;
  `,
  group: css`display:grid;gap:3px;`,
  groupLabel: css`
    padding: 3px 12px 5px;
    color: color-mix(in srgb,var(--sidebar-foreground) 48%,transparent);
    font-size: 12px;
    font-weight: 600;
    letter-spacing: .04em;
    line-height: 1.4;
  `,
  groupItems: css`display:grid;gap:2px;`,
  navLink: css`
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 9px 12px;
    font-size: 14px;
    font-weight: 500;
    color: color-mix(in srgb, var(--sidebar-foreground) 66%, transparent);
    border-radius: ${tk.radius}px;
    cursor: pointer;
    transition: color 0.2s ease, background 0.2s ease;
    border: none;
    background: transparent;
    white-space: nowrap;
    text-align: left;
    width: 100%;
    &:hover {
      color: var(--sidebar-foreground);
      background: var(--sidebar-accent);
    }
  `,
  navLinkCollapsed: css`
    padding: 11px;
    justify-content: center;
  `,
  navLinkActive: css`
    color: var(--sidebar-accent-foreground);
    background: var(--sidebar-accent);
    &:hover {
      color: var(--sidebar-accent-foreground);
      background: var(--sidebar-accent);
    }
  `,
  secondaryNav: css`display:grid;gap:2px;margin:1px 0 5px;padding:2px 4px 4px 34px;`,
  secondaryLink: css`
    position:relative;display:flex;align-items:center;gap:8px;width:100%;min-width:0;padding:8px 10px;
    border:0;border-radius:${tk.radiusSm}px;background:transparent;color:color-mix(in srgb,var(--sidebar-foreground) 62%,transparent);
    font-size:12px;line-height:1.35;text-align:left;cursor:pointer;
    &::before{content:'';position:absolute;left:-11px;top:0;bottom:0;width:1px;background:color-mix(in srgb,var(--sidebar-foreground) 15%,transparent);}
    &:hover{color:var(--sidebar-foreground);background:color-mix(in srgb,var(--sidebar-accent) 68%,transparent);}
  `,
  secondaryLinkActive: css`
    color:var(--sidebar-accent-foreground);font-weight:650;background:var(--sidebar-accent);
    &::before{width:2px;background:var(--sidebar-primary);}
  `,
  secondaryIndex: css`flex:0 0 18px;color:var(--sidebar-primary);font-variant-numeric:tabular-nums;`,
  secondaryLabel: css`min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;`,
}))

interface AppSidebarProps {
  collapsed: boolean
}

export default function AppSidebar({ collapsed }: AppSidebarProps) {
  const { styles, cx } = useStyles()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const projectMatch = location.pathname.match(/^\/games\/([^/]+)/)
  const activeProjectId = projectMatch?.[1]
  const activeGameStep = location.pathname.split('/')[3] || 'overview'

  const renderItem = (item: typeof NAV_ITEMS[number]) => {
    const active = location.pathname.startsWith(item.path)
    const IconComp = item.icon
    const button = (
      <button
        key={item.id}
        type="button"
        aria-label={collapsed ? t(item.label) : undefined}
        aria-current={active ? 'page' : undefined}
        className={cx(styles.navLink, collapsed && styles.navLinkCollapsed, active && styles.navLinkActive)}
        onClick={async () => { await navigate(item.path) }}
      >
        <IconComp size={18} weight={active ? 'fill' : 'regular'} />
        {!collapsed && t(item.label)}
      </button>
    )
    return collapsed ? <Tooltip key={item.id} title={t(item.label)} placement="right">{button}</Tooltip> : button
  }

  const renderGameSteps = () => activeProjectId ? <div className={styles.secondaryNav} aria-label="当前游戏制作步骤">
    {GAME_EDITOR_STEPS.map((step, index) => {
      const active = activeGameStep === step.key
      return <button
        type="button"
        key={step.key}
        aria-current={active ? 'step' : undefined}
        className={cx(styles.secondaryLink, active && styles.secondaryLinkActive)}
        onClick={() => { void navigate(`/games/${activeProjectId}/${step.key}`) }}
      >
        <span className={styles.secondaryIndex}>{index + 1}.</span>
        <span className={styles.secondaryLabel}>{step.label}</span>
      </button>
    })}
  </div> : null

  return (
    <aside className={cx(styles.sidebar, collapsed && styles.sidebarCollapsed)}>
      <div className={cx(styles.logo, collapsed && styles.logoCollapsed)}>
        <BrandMark size={30} />
        {!collapsed && <span className={styles.logoText}>Zerone Hub</span>}
      </div>

      <nav className={styles.nav}>
        {collapsed ? NAV_ITEMS.map(renderItem) : NAV_GROUPS.map((group) => (
          <section className={styles.group} key={group.id} aria-labelledby={`nav-group-${group.id}`}>
            <div className={styles.groupLabel} id={`nav-group-${group.id}`}>{group.label}</div>
            <div className={styles.groupItems}>{group.items.map((item) => <div key={item.id}>{renderItem(item)}{item.id === 'games' && renderGameSteps()}</div>)}</div>
          </section>
        ))}
      </nav>
      <ExtensionSidebarSection />
    </aside>
  )
}
