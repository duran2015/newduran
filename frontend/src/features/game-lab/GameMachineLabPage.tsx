import { useMemo, useState } from 'react'
import { Alert, Button, Empty, Skeleton, Switch, Tag, message } from 'antd'
import {
  ArrowRightIcon,
  CheckCircleIcon,
  FlaskIcon,
  PackageIcon,
  PlayIcon,
  WarningCircleIcon,
} from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import PrimaryButton from '@/components/PrimaryButton'
import { parseApiError } from '@/api/client'
import type {
  GameActionDefinition,
  GamePackageManifest,
  GamePackageSummary,
  GameStateDiff,
  GameTestRunResult,
} from '@/api/gamePackages'
import {
  useGamePackage,
  useGamePackages,
  useGamePackageTestRun,
  useInstallGameAcceptanceFixtures,
  useSetGamePackageEnabled,
} from '@/queries/useGamePackages'
import { tokens as t } from '@/styles/tokens'

const useStyles = createStyles(({ css }) => ({
  page: css`
    width: 100%; max-width: 1480px; margin: 0 auto;
  `,
  header: css`
    display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 22px;
    @media (max-width: 720px) { flex-direction: column; }
  `,
  title: css`
    margin: 0; color: ${t.text}; font-size: ${t.text2xl}; font-weight: 700; letter-spacing: -0.03em;
  `,
  subtitle: css`
    max-width: 74ch; margin: 6px 0 0; color: ${t.textTertiary}; font-size: ${t.textBase}; line-height: 1.65;
  `,
  stage: css`
    display: inline-flex; align-items: center; gap: 8px; flex-shrink: 0; padding: 9px 12px;
    border: 1px solid color-mix(in srgb, var(--success) 30%, var(--border)); border-radius: 999px;
    color: ${t.textSecondary}; background: color-mix(in srgb, var(--success) 7%, var(--card)); font-size: ${t.textSm};
  `,
  explainer: css`
    display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(260px, .65fr); gap: 16px; margin-bottom: 18px;
    padding: 18px 20px; border: 1px solid color-mix(in srgb, var(--primary) 22%, var(--border));
    border-radius: ${t.radiusLg}px; background: color-mix(in srgb, var(--primary) 5%, var(--card));
    @media (max-width: 760px) { grid-template-columns: 1fr; padding: 16px; }
  `,
  explainerTitle: css`
    margin: 0 0 7px; color: ${t.text}; font-size: ${t.textLg}; font-weight: 700;
  `,
  explainerText: css`
    margin: 0; color: ${t.textSecondary}; font-size: ${t.textSm}; line-height: 1.65;
  `,
  explainerSteps: css`
    display: grid; gap: 7px; align-content: center; color: ${t.textSecondary}; font-size: ${t.textSm};
    span { display: flex; align-items: center; gap: 8px; }
    b { display: inline-grid; place-items: center; width: 22px; height: 22px; border-radius: 50%; color: var(--primary); background: var(--card); }
  `,
  layout: css`
    display: grid; grid-template-columns: minmax(260px, 0.72fr) minmax(0, 1.8fr); min-height: 680px;
    overflow: hidden; border: 1px solid var(--border); border-radius: ${t.radiusLg}px; background: var(--card); box-shadow: ${t.elevation1};
    @media (max-width: 900px) { grid-template-columns: 1fr; }
  `,
  packagePane: css`
    padding: 18px; border-right: 1px solid var(--border); background: color-mix(in srgb, var(--background) 65%, var(--card));
    @media (max-width: 900px) { border-right: 0; border-bottom: 1px solid var(--border); }
  `,
  paneTitle: css`
    margin: 0 0 5px; color: ${t.text}; font-size: ${t.textLg}; font-weight: 650;
  `,
  paneHint: css`
    margin: 0 0 16px; color: ${t.textMuted}; font-size: ${t.textSm}; line-height: 1.5;
  `,
  packageList: css`
    display: grid; gap: 10px;
  `,
  packageButton: css`
    width: 100%; padding: 14px; border: 1px solid var(--border); border-radius: ${t.radius}px;
    color: ${t.text}; background: var(--card); text-align: left; cursor: pointer; transition: 0.18s ease;
    &:hover { border-color: color-mix(in srgb, var(--primary) 46%, var(--border)); transform: translateY(-1px); }
  `,
  packageActive: css`
    border-color: var(--primary); box-shadow: inset 3px 0 0 var(--primary); background: var(--primary-subtle);
  `,
  packageName: css`
    display: flex; align-items: center; justify-content: space-between; gap: 8px; font-weight: 650;
  `,
  packageDescription: css`
    display: -webkit-box; margin-top: 7px; overflow: hidden; color: ${t.textTertiary}; font-size: ${t.textSm}; line-height: 1.5;
    -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  `,
  packageCounts: css`
    display: flex; flex-wrap: wrap; gap: 8px; margin-top: 12px; color: ${t.textMuted}; font-size: 12px;
  `,
  detail: css`
    min-width: 0; padding: 24px;
    @media (max-width: 640px) { padding: 16px; }
  `,
  detailHeader: css`
    display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; padding-bottom: 20px; border-bottom: 1px solid var(--border);
    @media (max-width: 640px) { flex-direction: column; }
  `,
  detailTitle: css`
    margin: 0; color: ${t.text}; font-size: ${t.textXl}; font-weight: 700;
  `,
  detailDescription: css`
    max-width: 70ch; margin: 7px 0 0; color: ${t.textTertiary}; line-height: 1.6;
  `,
  toggle: css`
    display: flex; align-items: center; gap: 10px; flex-shrink: 0; color: ${t.textSecondary}; font-size: ${t.textSm};
  `,
  registry: css`
    display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin: 20px 0;
    @media (max-width: 720px) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  `,
  registryCard: css`
    min-width: 0; padding: 14px; border: 1px solid var(--border); border-radius: ${t.radius}px; background: var(--background);
  `,
  registryLabel: css`
    color: ${t.textMuted}; font-size: 12px;
  `,
  registryValue: css`
    margin-top: 5px; color: ${t.text}; font-size: ${t.textLg}; font-weight: 700;
  `,
  section: css`
    margin-top: 24px;
  `,
  sectionHeader: css`
    display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 12px;
  `,
  sectionTitle: css`
    margin: 0; color: ${t.text}; font-size: ${t.textLg}; font-weight: 650;
  `,
  tabs: css`
    display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px;
  `,
  tabButton: css`
    padding: 7px 11px; border: 1px solid var(--border); border-radius: 999px; color: ${t.textSecondary}; background: transparent; cursor: pointer;
  `,
  tabActive: css`
    color: var(--primary); border-color: var(--primary); background: var(--primary-subtle);
  `,
  definitions: css`
    display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 10px;
  `,
  definition: css`
    min-width: 0; padding: 13px 14px; border: 1px solid var(--border); border-radius: ${t.radiusSm}px; background: var(--card);
    strong { display: block; overflow-wrap: anywhere; color: ${t.text}; }
    span { display: block; margin-top: 5px; color: ${t.textTertiary}; font-size: ${t.textSm}; line-height: 1.5; }
  `,
  actions: css`
    display: flex; align-items: center; flex-wrap: wrap; gap: 10px;
  `,
  runResult: css`
    margin-top: 14px; padding: 16px; border: 1px solid var(--border); border-radius: ${t.radius}px; background: var(--background);
  `,
  runStatus: css`
    display: flex; align-items: center; gap: 8px; margin-bottom: 15px; color: ${t.text}; font-weight: 650;
  `,
  flow: css`
    display: grid; gap: 10px;
  `,
  step: css`
    display: grid; grid-template-columns: minmax(140px, 0.75fr) auto minmax(140px, 1fr) auto minmax(180px, 1.2fr); align-items: stretch; gap: 9px;
    @media (max-width: 760px) { grid-template-columns: 1fr; svg { transform: rotate(90deg); justify-self: center; } }
  `,
  flowBox: css`
    min-width: 0; padding: 12px; border: 1px solid var(--border); border-radius: ${t.radiusSm}px; background: var(--card);
  `,
  flowLabel: css`
    display: block; margin-bottom: 5px; color: ${t.textMuted}; font-size: 11px; font-weight: 650; letter-spacing: .04em; text-transform: uppercase;
  `,
  flowContent: css`
    color: ${t.text}; font-size: ${t.textSm}; line-height: 1.5; overflow-wrap: anywhere;
  `,
  diff: css`
    display: grid; gap: 5px; color: ${t.textSecondary}; font-size: ${t.textSm};
  `,
  loading: css`
    padding: 30px;
  `,
}))

type DefinitionTab = 'metrics' | 'actions' | 'events' | 'endings'

function isEnabled(pkg: GamePackageSummary) {
  return pkg.enabled ?? (pkg.status === 'enabled' || pkg.status === 'active')
}

function labelOf(item: { id: string; name?: string; displayName?: string }) {
  return item.displayName ?? item.name ?? item.id
}

function stringify(value: unknown) {
  if (value === undefined) return '—'
  if (typeof value === 'string') return value
  return JSON.stringify(value)
}

function normalizeDiff(diff: GameStateDiff[] | Record<string, unknown> | undefined): GameStateDiff[] {
  if (!diff) return []
  if (Array.isArray(diff)) return diff
  return Object.entries(diff).map(([key, value]) => {
    if (value && typeof value === 'object' && ('before' in value || 'after' in value)) {
      return { key, ...(value as GameStateDiff) }
    }
    return { key, after: value }
  })
}

function PackageList({ packages, selectedId, onSelect }: { packages: GamePackageSummary[]; selectedId?: string; onSelect: (id: string) => void }) {
  const { styles, cx } = useStyles()
  return (
    <div className={styles.packageList}>
      {packages.map((pkg) => (
        <button key={pkg.id} type="button" className={cx(styles.packageButton, pkg.id === selectedId && styles.packageActive)} onClick={() => { onSelect(pkg.id) }}>
          <div className={styles.packageName}>
            <span>{pkg.displayName}</span>
            <Tag color={isEnabled(pkg) ? 'success' : 'default'}>{isEnabled(pkg) ? '已启用' : '已停用'}</Tag>
          </div>
          <div className={styles.packageDescription}>{pkg.description ?? '这个游戏包暂未填写说明。'}</div>
          <div className={styles.packageCounts}>
            <Tag variant="filled">技术样例</Tag>
            <span>{pkg.registry.metricCount} 个状态项</span><span>·</span>
            <span>{pkg.registry.actionCount} 个可执行操作</span><span>·</span>
            <span>{pkg.registry.eventCount} 条系统记录</span>
          </div>
        </button>
      ))}
    </div>
  )
}

function ManifestDefinitions({ manifest }: { manifest: GamePackageManifest }) {
  const { styles, cx } = useStyles()
  const [tab, setTab] = useState<DefinitionTab>('metrics')
  const definitions = manifest.registry
  const labels: Record<DefinitionTab, string> = { metrics: '状态项', actions: '可执行操作', events: '系统记录', endings: '结束方式' }
  return (
    <>
      <div className={styles.tabs} role="tablist" aria-label="游戏包内容">
        {(Object.keys(labels) as DefinitionTab[]).map((key) => (
          <button key={key} type="button" role="tab" aria-selected={tab === key} className={cx(styles.tabButton, tab === key && styles.tabActive)} onClick={() => { setTab(key) }}>
            {labels[key]} {definitions[key].length}
          </button>
        ))}
      </div>
      {definitions[tab].length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={`暂未声明${labels[tab]}`} /> : (
        <div className={styles.definitions}>
          {definitions[tab].map((item) => <div className={styles.definition} key={item.id}><strong>{labelOf(item)}</strong><span>{item.description ?? '用于确认这类规则能够被游戏机器正确执行'}</span></div>)}
        </div>
      )}
    </>
  )
}

function TestRunResult({ result, manifest }: { result: GameTestRunResult; manifest: GamePackageManifest }) {
  const { styles } = useStyles()
  const actions = new Map(manifest.registry.actions.map((item) => [item.id, labelOf(item)]))
  const events = new Map(manifest.registry.events.map((item) => [item.id, labelOf(item)]))
  const metrics = new Map(manifest.registry.metrics.map((item) => [item.id, labelOf(item)]))
  return (
    <div className={styles.runResult} aria-live="polite">
      <div className={styles.runStatus}>
        {result.passed ? <CheckCircleIcon size={20} color="var(--success)" weight="fill" /> : <WarningCircleIcon size={20} color="var(--destructive)" weight="fill" />}
        {result.passed ? `验证通过：机器完成了 ${result.steps.length} 个步骤，结果与规则一致` : '验证未通过：下方标出了出问题的步骤'}
      </div>
      <div className={styles.flow}>
        {result.steps.map((step) => {
          const diffs = normalizeDiff(step.stateDiff)
          const eventId = typeof step.event === 'string' ? step.event : step.event?.id
          const event = eventId ? events.get(eventId) ?? '系统已记录本步结果' : '本步没有新记录'
          return <div className={styles.step} key={step.sequence}>
            <div className={styles.flowBox}><span className={styles.flowLabel}>第 {step.sequence} 步·测试操作</span><div className={styles.flowContent}>{actions.get(step.command.actionId) ?? '执行样例中的测试操作'}</div></div>
            <ArrowRightIcon size={18} color="var(--text-muted)" />
            <div className={styles.flowBox}><span className={styles.flowLabel}>系统记录</span><div className={styles.flowContent}>{event}</div></div>
            <ArrowRightIcon size={18} color="var(--text-muted)" />
            <div className={styles.flowBox}><span className={styles.flowLabel}>结果变化</span><div className={styles.diff}>
              {diffs.length ? diffs.map((diff, index) => {
                const metricId = diff.metricId ?? diff.path ?? diff.key
                return <span key={`${metricId}-${index}`}>{metricId ? metrics.get(metricId) ?? '状态数值' : '状态数值'}：{stringify(diff.before ?? diff.oldValue)} → {stringify(diff.after ?? diff.newValue)}</span>
              }) : <span>本步没有数值变化</span>}
            </div></div>
          </div>
        })}
      </div>
    </div>
  )
}

export default function GameMachineLabPage() {
  const { styles } = useStyles()
  const listQuery = useGamePackages()
  const packages = listQuery.data?.items ?? []
  const [requestedId, setRequestedId] = useState<string>()
  const selectedId = requestedId && packages.some((pkg) => pkg.id === requestedId) ? requestedId : packages[0]?.id
  const detailQuery = useGamePackage(selectedId)
  const toggleMutation = useSetGamePackageEnabled()
  const runMutation = useGamePackageTestRun()
  const fixturesMutation = useInstallGameAcceptanceFixtures()

  const detail = detailQuery.data
  const selected = detail?.package ?? packages.find((pkg) => pkg.id === selectedId)
  const firstAction = useMemo<GameActionDefinition | undefined>(() => detail?.manifest.registry.actions[0], [detail])
  const firstScenario = detail?.manifest.acceptanceScenarios?.[0]

  const toggle = async (enabled: boolean) => {
    if (!selected) return
    try {
      await toggleMutation.mutateAsync({ id: selected.id, enabled })
      message.success(enabled ? '游戏包已启用，可以开始测试' : '游戏包已停用，其他游戏包不受影响')
    } catch (error) { message.error(parseApiError(error)) }
  }

  const run = async () => {
    if (!selected || !firstAction) return
    try {
      await runMutation.mutateAsync({
        id: selected.id,
        commands: firstScenario?.commands ?? [{ actionId: firstAction.id, input: firstAction.exampleInput }],
        initialState: firstScenario?.initialState,
      })
    } catch (error) { message.error(parseApiError(error)) }
  }

  return <main className={styles.page}>
    <header className={styles.header}>
      <div><h1 className={styles.title}>游戏机器实验室</h1><p className={styles.subtitle}>这是底层能力验收页，不是 Speeding 游戏页面。这一阶段只确认：我们造的“游戏机器”能装入不同规则，而且不会互相影响。</p></div>
      <div className={styles.stage}><FlaskIcon size={17} weight="fill" color="var(--success)" />M0 · 游戏包与扩展验收</div>
    </header>

    <section className={styles.explainer} aria-labelledby="m0-explainer-title">
      <div><h2 id="m0-explainer-title" className={styles.explainerTitle}>为什么有“计数器”和“餐厅”？</h2><p className={styles.explainerText}>它们都是技术样例，不是我们要做的游戏题材。计数器用来验证最简单的数值变化；餐厅用来验证换成完全不同的规则后，底层仍然能工作。两者都通过，才证明后面可以安全装入 Speeding 的财富、人物和事件规则。</p></div>
      <div className={styles.explainerSteps}><span><b>1</b>选一个技术样例</span><span><b>2</b>点击“开始验证”</span><span><b>3</b>只看中文结果是否符合预期</span></div>
    </section>

    {listQuery.isError && <Alert type="error" showIcon title="暂时无法读取游戏包" description="请确认游戏机器服务已启动后重试。" action={<Button onClick={() => void listQuery.refetch()}>重新加载</Button>} />}
    {listQuery.isLoading ? <div className={styles.loading}><Skeleton active /></div> : !listQuery.isError && packages.length === 0 ? <Empty description="还没有准备技术样例。准备后即可开始验收底层机器。"><PrimaryButton loading={fixturesMutation.isPending} onClick={() => { void fixturesMutation.mutateAsync().catch((error: unknown) => { message.error(parseApiError(error)) }) }}>准备验收数据</PrimaryButton></Empty> : !listQuery.isError && <div className={styles.layout}>
      <aside className={styles.packagePane}><h2 className={styles.paneTitle}>技术验证样例</h2><p className={styles.paneHint}>这些内容只用于确认底层机器能运行不同规则，不是 Speeding 的正式故事。</p><PackageList packages={packages} selectedId={selectedId} onSelect={(id) => { setRequestedId(id); runMutation.reset() }} /></aside>
      <section className={styles.detail}>
        {detailQuery.isLoading ? <Skeleton active /> : detailQuery.isError ? <Alert type="error" showIcon title="无法打开这个游戏包" action={<Button onClick={() => void detailQuery.refetch()}>重试</Button>} /> : detail && selected ? <>
          <div className={styles.detailHeader}>
            <div><h2 className={styles.detailTitle}>{selected.displayName}</h2><p className={styles.detailDescription}>{selected.description ?? detail.manifest.package.description ?? '这个游戏包暂未填写说明。'}</p></div>
            <div className={styles.toggle}><span>{isEnabled(selected) ? '允许运行' : '已停用'}</span><Switch aria-label="启用游戏包" checked={isEnabled(selected)} loading={toggleMutation.isPending} onChange={(value) => void toggle(value)} /></div>
          </div>
          <div className={styles.registry}>
            {([['状态项', detail.manifest.registry.metrics.length], ['可执行操作', detail.manifest.registry.actions.length], ['系统记录', detail.manifest.registry.events.length], ['结束方式', detail.manifest.registry.endings.length]] as const).map(([label, count]) => <div className={styles.registryCard} key={label}><div className={styles.registryLabel}>{label}</div><div className={styles.registryValue}>{count}</div></div>)}
          </div>
          <section className={styles.section}><div className={styles.sectionHeader}><div><h3 className={styles.sectionTitle}>这个包为机器提供了什么</h3><p className={styles.paneHint}>这些是包的公开声明；停用或更换游戏包不会改变机器本身。</p></div></div><ManifestDefinitions manifest={detail.manifest} /></section>
          <section className={styles.section}>
            <div className={styles.sectionHeader}><div><h3 className={styles.sectionTitle}>运行一次验证</h3><p className={styles.paneHint}>看看机器是否依次完成测试操作、记下发生的事，并得出正确结果。</p></div></div>
            <div className={styles.actions}>
              <PrimaryButton icon={<PlayIcon size={16} weight="fill" />} disabled={!isEnabled(selected) || !firstAction} loading={runMutation.isPending} onClick={() => void run()}>{firstAction ? `开始验证：${firstScenario?.name ?? labelOf(firstAction)}` : '这个样例还没有可验证的操作'}</PrimaryButton>
              {!isEnabled(selected) && <span className={styles.paneHint}>先启用这个游戏包才能测试。</span>}
            </div>
            {runMutation.data && <TestRunResult result={runMutation.data} manifest={detail.manifest} />}
          </section>
        </> : <Empty image={<PackageIcon size={48} />} description="请选择一个游戏包" />}
      </section>
    </div>}
  </main>
}
