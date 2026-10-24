import { useEffect, useMemo, useState } from 'react'
import { Alert, App, Button, Empty, Result, Skeleton, Tag } from 'antd'
import { ArrowClockwiseIcon, CheckCircleIcon, PauseIcon, PlayIcon, StopIcon, UserCircleIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import type { GameProjectCharacter, GameProjectMetric, GameSession, GameSessionTimelineItem } from '@/api/gameProjects'
import PrimaryButton from '@/components/PrimaryButton'
import {
  useAdvanceGameSession,
  useCreateGameSession,
  useGameSession,
  useGameSessions,
  useTransitionGameSession,
} from '@/queries/useGameProjects'
import { tokens as t } from '@/styles/tokens'

interface Props {
  projectId: string
  metrics: GameProjectMetric[]
  characters: GameProjectCharacter[]
}

const statusCopy: Record<GameSession['status'], { label: string; color: string; message: string }> = {
  running: { label: '进行中', color: 'green', message: '可以继续推进下一回合' },
  paused: { label: '已暂停', color: 'orange', message: '当前进度已保存，继续后可接着推进' },
  completed: { label: '已结束', color: 'default', message: '这次试玩已经结束，可查看完整过程' },
}

const useStyles = createStyles(({ css }) => ({
  root: css`margin-top:24px;padding-top:24px;border-top:1px solid var(--border);`,
  heading: css`display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:16px;@media(max-width:720px){flex-direction:column;align-items:stretch;}`,
  title: css`margin:0;color:${t.text};font-size:${t.textXl};font-weight:650;`,
  intro: css`max-width:68ch;margin:6px 0 0;color:${t.textTertiary};font-size:${t.textBase};line-height:1.65;`,
  start: css`flex:0 0 auto;@media(max-width:720px){width:100%;height:44px;}`,
  empty: css`padding:36px 20px;border:1px dashed var(--border);border-radius:${t.radius}px;background:color-mix(in srgb,var(--background) 65%,var(--card));`,
  loading: css`padding:22px;border:1px solid var(--border);border-radius:${t.radius}px;`,
  statusBar: css`display:flex;align-items:center;justify-content:space-between;gap:14px;padding:14px 16px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);@media(max-width:680px){align-items:flex-start;flex-direction:column;}`,
  statusLeft: css`display:flex;align-items:center;gap:10px;min-width:0;flex-wrap:wrap;`,
  statusText: css`color:${t.textSecondary};`,
  controls: css`display:flex;align-items:center;gap:8px;flex-wrap:wrap;@media(max-width:520px){display:grid;grid-template-columns:1fr 1fr;width:100%;button{width:100%;height:44px;}}`,
  body: css`display:grid;grid-template-columns:minmax(0,1fr) minmax(300px,.72fr);gap:22px;margin-top:22px;@media(max-width:900px){grid-template-columns:1fr;}`,
  section: css`min-width:0;`,
  sectionTitle: css`margin:0 0 12px;color:${t.text};font-size:${t.textLg};font-weight:650;`,
  metrics: css`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);overflow:hidden;@media(max-width:520px){grid-template-columns:1fr;}`,
  metric: css`min-width:0;padding:15px 16px;border-bottom:1px solid var(--border);&:nth-child(odd){border-right:1px solid var(--border);}@media(min-width:521px){&:nth-last-child(-n+2){border-bottom:0;}}@media(max-width:520px){&:nth-child(odd){border-right:0;}&:last-child{border-bottom:0;}}`,
  metricName: css`display:block;color:${t.textTertiary};font-size:${t.textSm};`,
  metricValue: css`display:block;margin-top:5px;color:${t.text};font-size:${t.textXl};font-weight:650;font-variant-numeric:tabular-nums;`,
  event: css`padding:17px;border-left:3px solid var(--primary);border-radius:0 ${t.radius}px ${t.radius}px 0;background:var(--background);`,
  eventTop: css`display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;`,
  eventName: css`margin:0;color:${t.text};font-size:${t.textLg};font-weight:650;`,
  eventContent: css`margin:8px 0 0;color:${t.textSecondary};font-size:${t.textBase};line-height:1.7;white-space:pre-wrap;`,
  changes: css`display:flex;gap:7px;flex-wrap:wrap;margin-top:12px;`,
  change: css`padding:5px 9px;border-radius:999px;color:${t.textSecondary};font-size:${t.textSm};background:var(--secondary);`,
  people: css`display:flex;gap:8px;flex-wrap:wrap;margin-top:12px;`,
  person: css`display:inline-flex;align-items:center;gap:5px;color:${t.textSecondary};font-size:${t.textSm};`,
  timeline: css`border-left:2px solid color-mix(in srgb,var(--border) 85%,transparent);`,
  timelineItem: css`position:relative;margin-left:18px;padding:0 0 18px 8px;&:last-child{padding-bottom:0;}&::before{content:'';position:absolute;left:-26px;top:5px;width:10px;height:10px;border:2px solid var(--card);border-radius:50%;background:var(--primary);box-shadow:0 0 0 1px var(--border);}`,
  timelineName: css`margin:0;color:${t.text};font-weight:650;`,
  timelineText: css`margin:4px 0 0;color:${t.textTertiary};font-size:${t.textSm};line-height:1.55;`,
  timelineTime: css`display:block;margin-top:4px;color:${t.textMuted};font-size:12px;`,
  quiet: css`padding:24px 18px;border:1px dashed var(--border);border-radius:${t.radius}px;color:${t.textTertiary};text-align:center;`,
}))

function requestKey() {
  return globalThis.crypto?.randomUUID?.() ?? `turn-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function timelineDescription(item: GameSessionTimelineItem) {
  const output = item.output ?? {}
  if (typeof output.summary === 'string') return output.summary
  if (typeof output.description === 'string') return output.description
  const kindCopy: Record<string, string> = {
    session_started: '试玩开始，已载入初始世界状态。',
    'game.session.started': '试玩开始，已载入初始世界状态。',
    turn_advanced: '系统完成了本回合判断和数值结算。',
    'game.turn.advanced': '系统完成了本回合判断和数值结算。',
    event_triggered: '故事条件满足，事件已经发生。',
    'game.event.triggered': '故事条件满足，事件已经发生。',
    session_paused: '试玩已暂停，当前进度已保存。',
    'game.session.paused': '试玩已暂停，当前进度已保存。',
    session_resumed: '试玩继续进行。',
    'game.session.resumed': '试玩继续进行。',
    'game.session.running': '试玩继续进行。',
    session_completed: '试玩已结束。',
    'game.session.completed': '试玩已结束。',
  }
  return kindCopy[item.kind] ?? '系统记录了一次游戏进展。'
}

export default function GameSessionWorkspace({ projectId, metrics, characters }: Props) {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const sessionsQuery = useGameSessions(projectId)
  const [selectedId, setSelectedId] = useState<string>()
  const sessionQuery = useGameSession(projectId, selectedId)
  const createMutation = useCreateGameSession()
  const advanceMutation = useAdvanceGameSession()
  const pauseMutation = useTransitionGameSession('pause')
  const resumeMutation = useTransitionGameSession('resume')
  const endMutation = useTransitionGameSession('end')

  useEffect(() => {
    if (selectedId || !sessionsQuery.data?.length) return
    const latest = [...sessionsQuery.data].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))[0]
    setSelectedId(latest.id)
  }, [selectedId, sessionsQuery.data])

  const session = sessionQuery.data ?? sessionsQuery.data?.find((item) => item.id === selectedId)
  const director = characters.find((item) => item.id === session?.directorCharacterId) ?? characters.find((item) => item.characterType === 'director')
  const metricNames = useMemo(() => Object.fromEntries(metrics.map((item) => [item.id, item.name])), [metrics])
  const participantNames = useMemo(() => Object.fromEntries(characters.map((item) => [item.id, item.displayName])), [characters])
  const busy = createMutation.isPending || advanceMutation.isPending || pauseMutation.isPending || resumeMutation.isPending || endMutation.isPending

  const create = async () => {
    try {
      const next = await createMutation.mutateAsync(projectId)
      setSelectedId(next.id)
      message.success('试玩已开始')
    } catch {
      message.error('暂时无法开始试玩，请先保存规则、人物和事件后重试')
    }
  }

  const advance = async () => {
    if (!session) return
    try {
      await advanceMutation.mutateAsync({ projectId, sessionId: session.id, actorCharacterId: director?.id, requestKey: requestKey() })
      message.success('下一回合已完成')
    } catch {
      message.error('推进失败，请检查事件条件或稍后重试')
    }
  }

  const transition = async (action: 'pause' | 'resume' | 'end') => {
    if (!session) return
    try {
      const mutation = action === 'pause' ? pauseMutation : action === 'resume' ? resumeMutation : endMutation
      await mutation.mutateAsync({ projectId, sessionId: session.id, actorCharacterId: director?.id })
      message.success(action === 'pause' ? '试玩已暂停' : action === 'resume' ? '试玩已继续' : '试玩已结束')
    } catch {
      message.error('操作失败，请稍后重试')
    }
  }

  return <section className={styles.root} aria-labelledby="game-session-title">
    <div className={styles.heading}>
      <div><h3 className={styles.title} id="game-session-title">完整试玩</h3><p className={styles.intro}>从初始状态开始推进真实回合，检查事件是否按预期发生、数值是否正确变化，以及哪些人物参与了故事。</p></div>
      {session
        ? <Button className={styles.start} icon={<PlayIcon/>} loading={createMutation.isPending} onClick={() => void create()}>开始一局新试玩</Button>
        : <PrimaryButton className={styles.start} icon={<PlayIcon/>} loading={createMutation.isPending} onClick={() => void create()}>开始一局新试玩</PrimaryButton>}
    </div>

    {sessionsQuery.isLoading ? <div className={styles.loading}><Skeleton active paragraph={{ rows: 5 }}/></div>
      : sessionsQuery.isError ? <Result status="error" title="暂时无法读取试玩记录" subTitle="请检查网络后重试，已经保存的试玩不会丢失。" extra={<Button icon={<ArrowClockwiseIcon/>} onClick={() => void sessionsQuery.refetch()}>重试</Button>}/>
        : !session ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有完整试玩记录"><Button onClick={() => void create()}>开始第一局试玩</Button></Empty></div>
          : sessionQuery.isLoading ? <div className={styles.loading}><Skeleton active paragraph={{ rows: 6 }}/></div>
            : sessionQuery.isError ? <Alert showIcon type="error" title="这局试玩暂时打不开" description="请稍后重试，或开始一局新试玩。" action={<Button size="small" onClick={() => void sessionQuery.refetch()}>重试</Button>}/>
              : <>
                <div className={styles.statusBar}>
                  <div className={styles.statusLeft}><Tag color={statusCopy[session.status].color}>{statusCopy[session.status].label}</Tag><strong>{session.currentTurn === 0 ? '准备阶段' : `第 ${session.currentTurn} 回合`}</strong><span className={styles.statusText}>{statusCopy[session.status].message}</span></div>
                  <div className={styles.controls}>
                    {session.status === 'running' && <><PrimaryButton icon={<PlayIcon/>} loading={advanceMutation.isPending} disabled={busy} onClick={() => void advance()}>推进下一回合</PrimaryButton><Button icon={<PauseIcon/>} disabled={busy} onClick={() => void transition('pause')}>暂停</Button></>}
                    {session.status === 'paused' && <PrimaryButton icon={<PlayIcon/>} loading={resumeMutation.isPending} disabled={busy} onClick={() => void transition('resume')}>继续试玩</PrimaryButton>}
                    {session.status !== 'completed' && <Button icon={<StopIcon/>} disabled={busy} onClick={() => void transition('end')}>结束</Button>}
                  </div>
                </div>

                <div className={styles.body}>
                  <div className={styles.section}>
                    <h4 className={styles.sectionTitle}>当前局面</h4>
                    {metrics.length ? <div className={styles.metrics}>{metrics.map((metric) => <div className={styles.metric} key={metric.id}><span className={styles.metricName}>{metric.name}</span><span className={styles.metricValue}>{session.state[metric.id] ?? metric.initial}{metric.displayUnit ? ` ${metric.displayUnit}` : ''}</span></div>)}</div> : <div className={styles.quiet}>还没有可展示的世界状态</div>}

                    <h4 className={styles.sectionTitle} style={{ marginTop: 24 }}>刚刚发生</h4>
                    {session.latestEvent ? <article className={styles.event}>
                      <div className={styles.eventTop}><h5 className={styles.eventName}>{session.latestEvent.name}</h5>{session.latestEvent.endsGame && <Tag color="red">故事结束</Tag>}</div>
                      <p className={styles.eventContent}>{session.latestEvent.displayContent || '事件已经发生并完成结算。'}</p>
                      {session.latestEvent.stateDiff?.length ? <div className={styles.changes}>{session.latestEvent.stateDiff.map((change) => <span className={styles.change} key={`${change.metricId}-${String(change.before)}-${String(change.after)}`}>{metricNames[change.metricId] ?? '世界状态'}：{String(change.before)} → {String(change.after)}</span>)}</div> : null}
                      {session.latestEvent.participants?.length ? <div className={styles.people}>{session.latestEvent.participants.map((person) => {
                        const characterId = typeof person === 'string' ? person : person.characterId
                        const displayName = typeof person === 'string' ? participantNames[person] : person.displayName || participantNames[person.characterId]
                        return <span className={styles.person} key={characterId}><UserCircleIcon size={17}/>{displayName || '未命名人物'}</span>
                      })}</div> : null}
                    </article> : <div className={styles.quiet}>还没有事件发生，推进下一回合后可在这里查看。</div>}
                  </div>

                  <div className={styles.section}>
                    <h4 className={styles.sectionTitle}>完整过程</h4>
                    {session.timeline.length ? <div className={styles.timeline}>{[...session.timeline].reverse().map((item) => <article className={styles.timelineItem} key={item.id}><h5 className={styles.timelineName}>{item.name || (item.kind === 'event_triggered' ? '故事事件发生' : '游戏进展')}</h5><p className={styles.timelineText}>{timelineDescription(item)}</p><time className={styles.timelineTime} dateTime={item.occurredAt}>{new Date(item.occurredAt).toLocaleString('zh-CN', { hour12: false })}</time></article>)}</div> : <div className={styles.quiet}>开始推进后，每一步都会记录在这里。</div>}
                  </div>
                </div>
                {session.status === 'completed' && <Alert style={{ marginTop: 22 }} showIcon icon={<CheckCircleIcon/>} type="success" title="本次试玩已结束" description="可以根据完整过程调整规则和事件，然后开始新一局验证。"/>}
              </>}
  </section>
}
