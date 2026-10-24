import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, App, Button, Empty, Input, InputNumber, Result, Segmented, Select, Skeleton, Tag } from 'antd'
import { ArrowLeftIcon, CheckIcon, PaperPlaneTiltIcon, PlayIcon, SignOutIcon, SpeakerHighIcon, SpeakerSlashIcon, SparkleIcon, XIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import { useNavigate, useParams } from 'react-router'
import PrimaryButton from '@/components/PrimaryButton'
import { authFetchBlob } from '@/api/agent-chat'
import { parseApiError } from '@/api/client'
import type { GameProjectMetric } from '@/api/gameProjects'
import type { PlayerActionPlan, PlayerGameObjective, PlayerGameSession, PlayerGameSource, PlayerMessage, PlayerScene, PlayerStateItem } from '@/api/playerGames'
import { useGameProject } from '@/queries/useGameProjects'
import { useAcceptPlayerObjective, useConfirmPlayerAction, useCreatePlayerGameSession, useInterpretPlayerAction, usePlayerGameSession, usePublishedGame, useRejectPlayerAction, useRejectPlayerObjective, useResetPlayerGameSession } from '@/queries/usePlayerGames'
import { tokens as t } from '@/styles/tokens'

const useStyles = createStyles(({ css }) => ({
  shell: css`min-height:100dvh;color:#f6f1ea;background:#171412;`,
  header: css`position:sticky;z-index:20;top:0;display:flex;align-items:center;justify-content:space-between;gap:16px;min-height:64px;padding:10px clamp(16px,3vw,42px);border-bottom:1px solid rgb(255 255 255 / 10%);background:rgb(23 20 18 / 88%);backdrop-filter:blur(18px);@media(max-width:640px){gap:8px;padding:8px 10px;.ant-tag{display:none;}button{padding-inline:7px;}}`,
  gameName: css`min-width:0;display:flex;align-items:center;gap:10px;strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:${t.textLg};}.ant-tag{margin:0;}`,
  back: css`display:inline-flex;align-items:center;gap:6px;padding:8px;border:0;color:rgb(255 255 255 / 72%);background:transparent;cursor:pointer;&:focus-visible{outline:2px solid #ee8a62;outline-offset:2px;}`,
  stage: css`position:relative;min-height:calc(100dvh - 64px);overflow:hidden;`,
  backdrop: css`position:absolute;inset:0;background:radial-gradient(circle at 50% 28%,#574239 0,#221b18 50%,#151210 100%);img{width:100%;height:100%;object-fit:cover;opacity:.7;}.veil{position:absolute;inset:0;background:linear-gradient(180deg,rgb(15 12 10 / 24%),rgb(15 12 10 / 45%) 52%,#171412 92%);}`,
  portraits: css`position:absolute;z-index:1;inset:0 0 210px;overflow:hidden;pointer-events:none;@media(max-width:800px){inset:0 0 46%;}`,
  portrait: css`position:absolute;bottom:0;width:min(38vw,500px);height:94%;transform:translateX(-50%);transform-origin:bottom center;opacity:.32;filter:saturate(.55);transition:opacity .28s ease,filter .28s ease,transform .28s ease;&[data-active="true"]{opacity:1;filter:none;transform:translateX(-50%) scale(1.035);}img{width:100%;height:100%;object-fit:contain;object-position:bottom;filter:drop-shadow(0 12px 24px rgb(0 0 0 / 46%));}@media(max-width:800px){width:min(62vw,360px);height:88%;}@media(prefers-reduced-motion:reduce){transition:none;}`,
  portraitFallback: css`position:absolute;bottom:4%;display:grid;place-items:end center;width:clamp(150px,20vw,260px);height:clamp(260px,52vh,560px);transform:translateX(-50%);overflow:hidden;border:1px solid rgb(255 255 255 / 13%);border-radius:120px 120px 24px 24px;opacity:.28;filter:saturate(.4);background:linear-gradient(180deg,rgb(107 132 142 / 18%),rgb(20 18 17 / 82%) 52%,rgb(15 13 12 / 96%));box-shadow:inset 0 1px rgb(255 255 255 / 12%),0 26px 70px rgb(0 0 0 / 38%);transition:opacity .28s ease,filter .28s ease,transform .28s ease;&[data-active="true"]{opacity:1;filter:none;transform:translateX(-50%) scale(1.035);}&::before{content:'';position:absolute;top:13%;width:34%;aspect-ratio:1;border-radius:50%;background:linear-gradient(145deg,#746e69,#282524);box-shadow:0 20px 0 38px #302c2a;}.monogram{position:absolute;top:18%;z-index:1;color:rgb(255 255 255 / 74%);font:500 clamp(28px,4vw,50px)/1 serif}.nameplate{position:relative;z-index:2;width:100%;padding:18px 12px;text-align:center;background:linear-gradient(180deg,transparent,rgb(12 10 9 / 92%));strong{display:block;color:#fff;font-size:15px;letter-spacing:.12em}.role{display:block;margin-top:4px;color:#e7a27f;font-size:11px;letter-spacing:.08em}}`,
  content: css`position:relative;z-index:3;display:grid;grid-template-columns:minmax(0,1fr) 280px;gap:18px;width:min(1180px,calc(100% - 32px));min-height:calc(100dvh - 64px);margin:auto;padding:clamp(32px,8vh,84px) 0 24px;@media(max-width:800px){display:flex;flex-direction:column;justify-content:flex-end;width:min(100% - 24px,680px);padding-top:min(32vh,260px);}`,
  story: css`align-self:end;min-width:0;`,
  sceneMeta: css`margin-bottom:14px;text-shadow:0 2px 12px #000;.eyebrow{color:#f4aa83;font-size:${t.textSm};font-weight:700;letter-spacing:.08em;text-transform:uppercase;}h1{margin:5px 0 0;color:#fff;font-size:clamp(26px,5vw,48px);letter-spacing:-.035em;}p{max-width:68ch;margin:8px 0 0;color:rgb(255 255 255 / 74%);line-height:1.65;}`,
  dialogue: css`overflow:hidden;border:1px solid rgb(255 255 255 / 15%);border-radius:18px;background:rgb(13 12 12 / 88%);box-shadow:0 22px 60px rgb(0 0 0 / 42%);backdrop-filter:blur(18px);`,
  npcTabs: css`display:flex;gap:8px;margin:0 0 10px;overflow-x:auto;padding:2px 2px 5px;scrollbar-width:thin;`,
  npcChooser: css`margin:0 0 10px;.chooserHead{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:7px;color:rgb(255 255 255 / 68%);font-size:12px}.chooserHead strong{color:#fff;font-size:13px}.chooserHead span{color:rgb(255 255 255 / 48%);text-align:right}@media(max-width:520px){.chooserHead{align-items:flex-start;flex-direction:column;gap:3px}.chooserHead span{text-align:left}}`,
  npcTab: css`display:flex;flex:0 0 auto;align-items:center;gap:9px;padding:7px 12px 7px 8px;border:1px solid rgb(255 255 255 / 12%);border-radius:999px;color:rgb(255 255 255 / 62%);background:rgb(17 15 14 / 72%);cursor:pointer;&[aria-pressed="true"]{border-color:#e99a72;color:#fff;background:rgb(112 55 34 / 74%);box-shadow:0 0 0 2px rgb(233 154 114 / 12%);}.avatar{display:grid;place-items:center;width:28px;height:28px;border-radius:50%;color:#2b1710;font-weight:800;background:#f0b08d;}small{display:block;color:rgb(255 255 255 / 48%);font-size:10px;}&:focus-visible{outline:2px solid #f0a17b;outline-offset:2px;}`,
  dialogueHead: css`display:flex;align-items:center;justify-content:space-between;gap:12px;padding:11px 16px;border-bottom:1px solid rgb(255 255 255 / 10%);background:linear-gradient(90deg,rgb(143 76 48 / 52%),rgb(31 25 22 / 32%));.identity{display:flex;align-items:baseline;gap:9px}.name{color:#fff;font-size:17px;font-weight:750;letter-spacing:.04em}.status{color:#f0b08d;font-size:11px}.privacy{color:rgb(255 255 255 / 44%);font-size:11px;}`,
  dialogueBody: css`padding:14px 16px 16px;`,
  objective: css`display:flex;align-items:center;gap:10px;margin-bottom:12px;padding:9px 12px;border-left:2px solid #e88b63;color:rgb(255 255 255 / 66%);font-size:12px;letter-spacing:.03em;background:linear-gradient(90deg,rgb(232 139 99 / 10%),transparent);strong{color:#f6c0a5;font-weight:600;}`,
  messages: css`display:flex;flex-direction:column;gap:10px;max-height:290px;margin-bottom:14px;overflow:auto;scrollbar-width:thin;`,
  message: css`align-self:flex-start;max-width:min(82%,680px);padding:10px 13px;border-radius:4px 14px 14px 14px;background:rgb(255 255 255 / 7%);&[data-kind="player"]{align-self:flex-end;border-radius:14px 4px 14px 14px;background:rgb(139 73 46 / 72%);}.speaker{margin-bottom:3px;color:#f4aa83;font-size:11px;font-weight:700;}.text{color:rgb(255 255 255 / 92%);line-height:1.62;white-space:pre-wrap;}`,
  inputRow: css`display:flex;align-items:flex-end;gap:10px;.ant-input{color:#fff!important;border-color:rgb(255 255 255 / 14%)!important;background:rgb(255 255 255 / 7%)!important;resize:none;}.ant-input::placeholder{color:rgb(255 255 255 / 40%);}@media(max-width:520px){gap:7px;button{width:auto;min-width:78px;}}`,
  hint: css`margin:9px 0 0;color:rgb(255 255 255 / 48%);font-size:12px;line-height:1.5;`,
  side: css`align-self:end;display:grid;gap:12px;@media(max-width:800px){align-self:stretch;grid-template-columns:repeat(2,minmax(230px,1fr));overflow-x:auto;padding-bottom:4px;>button{grid-column:1/-1;justify-self:start;}.ant-btn{position:sticky;left:0;} }@media(max-width:520px){grid-template-columns:repeat(3,minmax(250px,82vw));}`,
  panel: css`padding:15px;border:1px solid rgb(255 255 255 / 12%);border-radius:16px;background:rgb(28 23 20 / 78%);backdrop-filter:blur(16px);h2{margin:0 0 11px;font-size:${t.textBase};}`,
  states: css`display:grid;gap:7px;`,
  state: css`display:flex;align-items:baseline;justify-content:space-between;gap:12px;padding:7px 0;border-bottom:1px solid rgb(255 255 255 / 8%);&:last-child{border:0;}span{color:rgb(255 255 255 / 63%);font-size:${t.textSm};}strong{font-variant-numeric:tabular-nums;}.change{margin-left:6px;color:#83d6ad;font-size:11px;}`,
  taskList: css`display:grid;gap:9px;`,
  task: css`padding:10px 11px;border:1px solid rgb(255 255 255 / 9%);border-radius:11px;background:rgb(255 255 255 / 3%);.taskTop{display:flex;align-items:flex-start;justify-content:space-between;gap:10px}.taskName{color:rgb(255 255 255 / 82%);font-size:13px;font-weight:650}.taskStatus{flex:0 0 auto;font-size:11px;font-weight:650}.issuer{display:block;margin-top:3px;color:rgb(255 255 255 / 43%);font-size:11px}.description{margin:7px 0 0;color:rgb(255 255 255 / 52%);font-size:12px;line-height:1.5}&[data-status="offered"]{border-color:rgb(244 170 131 / 42%);background:rgb(244 170 131 / 8%)}&[data-status="offered"] .taskStatus{color:#f4aa83}&[data-status="completed"] .taskStatus{color:#74d6b2}&[data-status="active"] .taskStatus{color:#f5c3aa}&[data-status="failed"]{border-color:rgb(238 116 106 / 26%)}&[data-status="failed"] .taskStatus{color:#ef8a80}&[data-status="locked"],&[data-status="rejected"]{opacity:.58}&[data-status="locked"] .taskStatus,&[data-status="rejected"] .taskStatus{color:rgb(255 255 255 / 45%)}`,
  offer: css`margin-bottom:14px;padding:16px;border:1px solid rgb(244 170 131 / 48%);border-radius:16px;background:linear-gradient(135deg,rgb(139 73 46 / 34%),rgb(35 27 23 / 92%));box-shadow:0 12px 34px rgb(0 0 0 / 20%);.offerFrom{color:#f4aa83;font-size:12px;font-weight:700;letter-spacing:.04em}.offerTitle{margin:5px 0 7px;color:#fff;font-size:18px}.offerDescription{margin:0;color:rgb(255 255 255 / 72%);font-size:13px;line-height:1.6}.offerFacts{display:grid;gap:6px;margin:12px 0 0;padding:10px 12px;border-radius:10px;background:rgb(0 0 0 / 18%);span{color:rgb(255 255 255 / 68%);font-size:12px;line-height:1.5}strong{color:#f7c6ad}.offerActions{display:flex;gap:8px;margin-top:13px}@media(max-width:460px){padding:14px;.offerActions{flex-direction:column}.offerActions button{width:100%}}`,
  plan: css`margin-bottom:12px;padding:14px;border:1px solid rgb(244 170 131 / 36%);border-radius:14px;background:rgb(244 170 131 / 9%);h3{display:flex;align-items:center;gap:7px;margin:0 0 7px;font-size:${t.textBase};}p{margin:0;color:rgb(255 255 255 / 84%);line-height:1.55;}ul{margin:8px 0 0;padding-left:20px;color:rgb(255 255 255 / 68%);line-height:1.55;}`,
  planActions: css`display:flex;gap:8px;margin-top:12px;@media(max-width:460px){flex-direction:column;button{width:100%;}}`,
  start: css`position:relative;z-index:2;display:grid;place-items:center;min-height:calc(100dvh - 64px);padding:24px;`,
  startCard: css`width:min(620px,100%);padding:clamp(24px,5vw,44px);border:1px solid rgb(255 255 255 / 13%);border-radius:24px;text-align:center;background:rgb(28 23 20 / 86%);box-shadow:0 24px 80px rgb(0 0 0 / 40%);h1{margin:8px 0;color:#fff;font-size:clamp(30px,6vw,52px);}p{margin:0 auto 24px;color:rgb(255 255 255 / 65%);line-height:1.7;}.draft{color:#f4aa83;font-size:13px;font-weight:700;}`,
  startOptions: css`display:grid;gap:10px;margin:0 0 22px;padding:14px;border:1px solid rgb(255 255 255 / 10%);border-radius:14px;text-align:left;background:rgb(255 255 255 / 4%);label{color:rgb(255 255 255 / 65%);font-size:13px;}.ant-segmented{width:100%;}.ant-segmented-group{display:grid;grid-template-columns:repeat(3,1fr);}.ant-input-number,.ant-select{width:100%;}`,
  result: css`.ant-result-title,.ant-result-subtitle{color:#fff!important;}`,
  thinking: css`display:flex;align-items:center;gap:10px;margin:8px 0 12px;padding:10px 12px;border-radius:12px;color:#f5c3aa;font-size:13px;background:rgb(232 139 99 / 9%);.pulse{width:8px;height:8px;border-radius:50%;background:#ee8a62;box-shadow:0 0 0 0 rgb(238 138 98 / 50%);animation:pulse 1.4s infinite}@keyframes pulse{70%{box-shadow:0 0 0 8px transparent}100%{box-shadow:0 0 0 0 transparent}}@media(prefers-reduced-motion:reduce){.pulse{animation:none}}`,
}))

function requestKey() { return crypto.randomUUID() }
const FREEFORM_INTERACTION_ID = '__freeform_interaction__'

function useProtectedUrls(urls: string[]) {
  const [resolved, setResolved] = useState<Record<string, string>>({})
  useEffect(() => {
    let active = true
    const objectUrls: string[] = []
    void Promise.all(urls.map(async (url) => {
      if (!url.startsWith('/api/')) { if (active) setResolved((value) => ({ ...value, [url]: url })); return }
      try {
        const blob = await authFetchBlob(url)
        const objectUrl = URL.createObjectURL(blob); objectUrls.push(objectUrl)
        if (active) setResolved((value) => ({ ...value, [url]: objectUrl }))
      } catch { /* The scene keeps a readable fallback when a resource cannot load. */ }
    }))
    return () => { active = false; objectUrls.forEach(URL.revokeObjectURL) }
  }, [urls.join('|')])
  return resolved
}

function PlanCard({ plan, busy, labels, onConfirm, onReject }: { plan: PlayerActionPlan; busy: boolean; labels: Record<string, string>; onConfirm: () => void; onReject: () => void }) {
  const { styles } = useStyles()
  return <div className={styles.plan} aria-live="polite">
    <h3><SparkleIcon weight="fill"/>系统整理出的行动</h3>
    <p>{plan.summary}</p>
    {(plan.steps?.length ?? 0) > 0 && <ul>{plan.steps?.map((step) => <li key={step}>{step}</li>)}</ul>}
    {(plan.effects?.length ?? 0) > 0 && <ul>{plan.effects?.map((effect) => <li key={effect.metricId}>{labels[effect.metricId] ?? effect.metricId}：{String(effect.before)} → {String(effect.after)}</li>)}</ul>}
    {plan.warnings.map((warning) => <Alert key={warning} style={{ marginTop: 9 }} type="warning" showIcon title={warning}/>)}
    <div className={styles.planActions}>
      {plan.needsConfirmation && <PrimaryButton icon={<CheckIcon/>} loading={busy} onClick={onConfirm}>确认执行</PrimaryButton>}
      <Button icon={<XIcon/>} disabled={busy} onClick={onReject}>重新说</Button>
    </div>
  </div>
}

function ObjectiveOffer({ objective, busy, error, onAccept, onReject }: { objective: PlayerGameObjective; busy: boolean; error?: string; onAccept: () => void; onReject: () => void }) {
  const { styles } = useStyles()
  return <section className={styles.offer} aria-label={`${objective.issuer || '有人'}提出的任务`}>
    <div className="offerFrom">{objective.issuer ? `${objective.issuer}向你提出一项任务` : '你收到一项新任务'}</div>
    <h3 className="offerTitle">{objective.name}</h3>
    {objective.description && <p className="offerDescription">{objective.description}</p>}
    <div className="offerFacts">
      {objective.promiseText && <span><strong>接受即代表承诺：</strong>{objective.promiseText}</span>}
      {objective.deadlineTurn ? <span><strong>期限：</strong>请在第 {objective.deadlineTurn} 回合结束前完成，逾期可能触发任务中约定的后果。</span> : <span><strong>期限：</strong>接受后开始计时；如有截止回合，会立即显示。</span>}
      {objective.description && <span><strong>利害：</strong>接受后将按上述目标结算；拒绝会留下记录，但不算任务失败。</span>}
    </div>
    {error && <Alert style={{ marginTop: 10 }} type="error" showIcon title="这次选择还没有生效" description={`${error}，请再试一次。`}/>} 
    <div className="offerActions"><PrimaryButton icon={<CheckIcon/>} loading={busy} disabled={busy} onClick={onAccept}>接受任务</PrimaryButton><Button icon={<XIcon/>} disabled={busy} onClick={onReject}>拒绝这次请求</Button></div>
  </section>
}

function playerView(session: PlayerGameSession, metrics: GameProjectMetric[]) {
  const metricMap = new Map(metrics.map((metric) => [metric.id, metric]))
  const scene: PlayerScene | undefined = session.scenes?.find((item) => item.id === session.currentSceneId) ?? session.scenes?.[0]
  const stateSummary: PlayerStateItem[] = Object.entries(session.state).map(([id, value]) => ({ id, name: session.stateLabels?.[id] ?? metricMap.get(id)?.name ?? id, value: `${String(value)}${metricMap.get(id)?.displayUnit ?? ''}` }))
  const messages: PlayerMessage[] = session.timeline.map((item, index) => ({ id: `${item.occurredAt}-${index}`, speaker: item.speaker ?? item.label, kind: item.speaker === '你' ? 'player' : item.speaker ? 'npc' : 'system', content: item.content, occurredAt: item.occurredAt, characterId: item.characterId, targetCharacterId: item.targetCharacterId }))
  return { scene, stateSummary, messages }
}

function SessionView({ session, metrics, releaseId }: { session: PlayerGameSession; metrics: GameProjectMetric[]; releaseId?: string }) {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const [text, setText] = useState('')
  const [muted, setMuted] = useState(true)
  const [activeCharacterId, setActiveCharacterId] = useState('')
  const listRef = useRef<HTMLDivElement>(null)
  const interpret = useInterpretPlayerAction()
  const confirm = useConfirmPlayerAction()
  const reject = useRejectPlayerAction()
  const acceptObjective = useAcceptPlayerObjective()
  const rejectObjective = useRejectPlayerObjective()
  const [objectiveDecisionError, setObjectiveDecisionError] = useState('')
  const { scene, stateSummary, messages } = useMemo(() => playerView(session, metrics), [session, metrics])
  const sceneIndex = Math.max(0, (session.scenes ?? []).findIndex((item) => item.id === scene?.id))
  const urls = useMemo(() => [scene?.backgroundUrl, scene?.musicUrl, ...(scene?.soundUrls ?? []), ...(scene?.characters.map((item) => item.portraitUrl) ?? [])].filter((url): url is string => Boolean(url)), [scene])
  const resolved = useProtectedUrls(urls)
  const musicUrl = scene?.musicUrl ? resolved[scene.musicUrl] : undefined
  const soundUrls = (scene?.soundUrls ?? []).map((url) => resolved[url]).filter(Boolean)
  const soundKey = soundUrls.join('|')
  const objectiveBusy = acceptObjective.isPending || rejectObjective.isPending
  const busy = interpret.isPending || confirm.isPending || reject.isPending || objectiveBusy
  const visibleCharacters = useMemo(() => (scene?.characters ?? []).filter((character) => character.visible !== false), [scene])
  const activeCharacter = visibleCharacters.find((character) => character.characterId === activeCharacterId) ?? visibleCharacters[0]
  const conversation = useMemo(() => {
    if (!activeCharacter) return []
    return messages.filter((item) => item.characterId === activeCharacter.characterId || (item.kind === 'player' && (item.targetCharacterId === activeCharacter.characterId || (!item.targetCharacterId && item.content.includes(activeCharacter.name)))))
  }, [activeCharacter, messages])
  const objectives = session.objectives ?? []
  const offeredObjective = objectives.find((item) => item.status === 'offered')
  const currentObjective = objectives.find((item) => item.status === 'active')
	const lastConversationItem = conversation.at(-1)
	const lastPlayerLine = [...conversation].reverse().find((item) => item.kind === 'player')
	const canRetryResponse = Boolean(lastPlayerLine && lastConversationItem?.kind === 'npc' && /(超时|未能取得|暂时无法回应)/.test(lastConversationItem.content))
	const review = useMemo(() => {
		const choices = session.timeline.filter((item) => item.label === '执行玩家行动' || item.label === '行动已执行')
		const changes = session.timeline.flatMap((item) => item.stateDiff ?? [])
		const resultByMetric = new Map<string, { before: unknown; after: unknown }>()
		for (const change of changes) {
			const current = resultByMetric.get(change.metricId)
			resultByMetric.set(change.metricId, { before: current?.before ?? change.before, after: change.after })
		}
		return { choices: choices.slice(-6), changes: [...resultByMetric.entries()] }
	}, [session.timeline])

  useEffect(() => {
    if (!activeCharacterId || !visibleCharacters.some((character) => character.characterId === activeCharacterId)) setActiveCharacterId(visibleCharacters[0]?.characterId ?? '')
  }, [activeCharacterId, visibleCharacters])

  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [messages.length, session.pendingPlan])
  useEffect(() => {
    if (muted || !musicUrl) return
    const audio = new Audio(musicUrl); audio.loop = true; audio.volume = .32
    void audio.play().catch(() => undefined)
    return () => { audio.pause() }
  }, [muted, musicUrl])
  useEffect(() => {
    if (muted || soundUrls.length !== (scene?.soundUrls?.length ?? 0)) return
    const sounds = soundUrls.map((url) => { const sound = new Audio(url); sound.volume = .5; void sound.play().catch(() => undefined); return sound })
    return () => sounds.forEach((sound) => sound.pause())
  }, [muted, scene?.id, soundKey])

  const sendText = async (raw: string) => {
		const value = raw.trim()
    if (!value || busy) return
    try {
      const interpreted = await interpret.mutateAsync({ projectId: session.projectId, sessionId: session.id, releaseId, text: value, targetCharacterId: activeCharacter?.characterId, requestKey: requestKey() })
      setText('')
      // Dialogue is a conversational turn, not a world-state command. Send it
      // straight to the characters; only rule-backed state changes need a
      // separate player confirmation.
      if (interpreted.pendingPlan?.actionId === FREEFORM_INTERACTION_ID) {
        await confirm.mutateAsync({ projectId: session.projectId, sessionId: session.id, releaseId, planId: interpreted.pendingPlan.id, requestKey: requestKey() })
      }
    } catch (error) { message.error(parseApiError(error)) }
  }
	const submit = async () => sendText(text)
  const accept = async () => {
    if (!session.pendingPlan) return
    try { await confirm.mutateAsync({ projectId: session.projectId, sessionId: session.id, releaseId, planId: session.pendingPlan.id, requestKey: requestKey() }) }
    catch (error) { message.error(parseApiError(error)) }
  }
  const decline = async () => {
    if (!session.pendingPlan) return
    try { await reject.mutateAsync({ projectId: session.projectId, sessionId: session.id, releaseId, planId: session.pendingPlan.id }) }
    catch (error) { message.error(parseApiError(error)) }
  }
  const decideObjective = async (objective: PlayerGameObjective, acceptOffer: boolean) => {
    if (objectiveBusy) return
    setObjectiveDecisionError('')
    const input = { projectId: session.projectId, sessionId: session.id, releaseId, objectiveId: objective.id, requestKey: requestKey() }
    try {
      if (acceptOffer) await acceptObjective.mutateAsync(input)
      else await rejectObjective.mutateAsync(input)
    } catch (error) { setObjectiveDecisionError(parseApiError(error)) }
  }

  return <div className={styles.stage}>
    <div className={styles.backdrop}>{scene?.backgroundUrl && resolved[scene.backgroundUrl] && <img src={resolved[scene.backgroundUrl]} alt=""/>}<div className="veil"/></div>
    <div className={styles.portraits}>{scene?.characters.map((character) => {
      if (character.visible === false) return null
      const x = character.x ?? 50
      const y = character.y ?? 0
      const scale = character.scale ?? 1
      const layer = character.layer ?? 1
      if (character.portraitUrl && resolved[character.portraitUrl]) return <div key={character.characterId} data-active={activeCharacter?.characterId === character.characterId} className={styles.portrait} style={{ left: `${x}%`, bottom: `${y}%`, scale: `${scale}`, zIndex: layer }}><img src={resolved[character.portraitUrl]} alt={character.name}/></div>
      return <div key={character.characterId} data-active={activeCharacter?.characterId === character.characterId} className={styles.portraitFallback} style={{ left: `${x}%`, scale: `${scale}`, zIndex: layer }} aria-label={`${character.name}在场`}><span className="monogram">{character.name.slice(0, 1)}</span><div className="nameplate"><strong>{character.name}</strong><span className="role">正在参与当前场景</span></div></div>
    })}</div>
    <div className={styles.content}>
      <section className={styles.story}>
        <div className={styles.sceneMeta}><span className="eyebrow">第 {sceneIndex + 1}/{Math.max(1, session.scenes?.length ?? 0)} 幕 · 第 {session.currentTurn} 回合</span><h1>{scene?.name ?? '故事正在继续'}</h1>{scene?.description && <p>{scene.description}</p>}</div>
        <div className={styles.npcChooser}>
          <div className="chooserHead"><strong>对话对象</strong><span>{visibleCharacters.length > 1 ? '点击姓名切换交谈对象' : `当前只有${activeCharacter?.name ?? '一名人物'}在场，其他人物会随剧情进入`}</span></div>
          <div className={styles.npcTabs} aria-label="选择对话人物">{visibleCharacters.map((character) => <button type="button" key={character.characterId} className={styles.npcTab} aria-pressed={activeCharacter?.characterId === character.characterId} onClick={() => setActiveCharacterId(character.characterId)}><span className="avatar">{character.name.slice(0, 1)}</span><span>{character.name}<small>{activeCharacter?.characterId === character.characterId ? '当前对话' : '点击交谈'}</small></span></button>)}</div>
        </div>
        <div className={styles.dialogue}>
          <div className={styles.dialogueHead}><div className="identity"><span className="name">{activeCharacter?.name ?? '当前场景'}</span><span className="status">{busy ? '正在思考…' : '在场'}</span></div><span className="privacy">这是你与该人物的对话</span></div>
          <div className={styles.dialogueBody}>
          <div className={styles.objective}><strong>{currentObjective ? '当前任务' : '当前目标'}</strong><span>{currentObjective?.name ?? session.game?.description ?? '自由探索当前局势'}</span>{currentObjective?.issuer && <small>来自 {currentObjective.issuer}</small>}</div>
          <div ref={listRef} className={styles.messages} aria-label="故事对话">
            {conversation.length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: 'rgb(255 255 255 / 54%)' }}>你还没有和{activeCharacter?.name ?? '这个人物'}说过话</span>}/>
              : conversation.map((item) => <article key={item.id} data-kind={item.kind} className={styles.message}><div className="speaker">{item.speaker}</div><div className="text">{item.content}</div></article>)}
          </div>
          {offeredObjective && session.status !== 'completed' && <ObjectiveOffer objective={offeredObjective} busy={objectiveBusy} error={objectiveDecisionError} onAccept={() => void decideObjective(offeredObjective, true)} onReject={() => void decideObjective(offeredObjective, false)}/>} 
          {session.pendingPlan && session.status !== 'completed' && <PlanCard plan={session.pendingPlan} busy={busy} labels={{ ...Object.fromEntries(metrics.map((metric) => [metric.id, metric.name])), ...session.stateLabels }} onConfirm={() => void accept()} onReject={() => void decline()}/>} 
          {busy && <div className={styles.thinking} aria-live="polite"><span className="pulse"/>人物正在根据各自掌握的信息判断局势……</div>}
		  {canRetryResponse && !busy && <Alert type="warning" showIcon title={`${activeCharacter?.name ?? '人物'}刚才没有成功回应`} description={<Button size="small" onClick={() => void sendText(lastPlayerLine?.content ?? '')}>重新询问这句话</Button>}/>} 
          {!session.pendingPlan && session.status !== 'completed' && <><div className={styles.inputRow}><Input.TextArea aria-label={`对${activeCharacter?.name ?? '人物'}说`} value={text} autoSize={{ minRows: 2, maxRows: 5 }} maxLength={1000} placeholder={activeCharacter ? `对${activeCharacter.name}说……` : '选择一位在场人物'} disabled={busy || !activeCharacter} onChange={(event) => setText(event.target.value)} onPressEnter={(event) => { if (!event.shiftKey) { event.preventDefault(); void submit() } }}/><PrimaryButton icon={<PaperPlaneTiltIcon/>} loading={busy} disabled={!text.trim() || !activeCharacter} onClick={() => void submit()}>发送</PrimaryButton></div><p className={styles.hint}>{busy ? `${activeCharacter?.name ?? '人物'}正在根据已知信息回应你……` : '你可以自由说话或提出行动；只有会改变世界状态的行动才需要确认。'}</p></>}
		  {session.status === 'completed' && <div className={styles.panel} style={{ marginTop: 16 }}><Alert type="success" showIcon title={session.ending?.title ?? '本局已经结束'} description={session.ending?.summary ?? '本局故事已经结束。'}/><h2 style={{ marginTop: 18 }}>本局复盘</h2><p>你做出了 {review.choices.length} 个关键决定，并改变了 {review.changes.length} 项重要局面。</p>{review.choices.length > 0 && <div className={styles.taskList}>{review.choices.map((item, index) => <article className={styles.task} key={`${item.occurredAt}-${index}`}><strong>{index + 1}. {item.content}</strong></article>)}</div>}{review.changes.length > 0 && <><h3 style={{ marginTop: 16 }}>最终影响</h3><div className={styles.states}>{review.changes.map(([metricId, change]) => <div className={styles.state} key={metricId}><span>{session.stateLabels?.[metricId] ?? metrics.find((metric) => metric.id === metricId)?.name ?? metricId}</span><strong>{String(change.before)} → {String(change.after)}</strong></div>)}</div></>}<h3 style={{ marginTop: 16 }}>任务结果</h3><div className={styles.taskList}>{objectives.map((item) => <article className={styles.task} key={item.id} data-status={item.status}><span>{item.name}</span><strong>{{ offered: '未决定', active: '未完成', completed: '已完成', failed: '已失败', rejected: '已拒绝', locked: '未解锁' }[item.status]}</strong></article>)}</div></div>}
          <details style={{ marginTop: 16 }} open={session.status === 'completed'}><summary style={{ cursor: 'pointer' }}>查看剧情和系统记录（{session.timeline.length} 条）</summary><div className={styles.messages} style={{ marginTop: 12 }} aria-label="本局复盘">{session.timeline.map((item, index) => <article className={styles.message} key={`${item.occurredAt}-${index}`}><div className="speaker">{item.speaker ?? item.label}</div><div className="text">{item.content}</div>{item.stateDiff?.map((diff) => <div key={diff.metricId}>{session.stateLabels?.[diff.metricId] ?? metrics.find((metric) => metric.id === diff.metricId)?.name ?? diff.metricId}：{String(diff.before)} → {String(diff.after)}</div>)}</article>)}</div></details>
          </div>
        </div>
      </section>
      <aside className={styles.side}>
        <Button icon={muted ? <SpeakerSlashIcon/> : <SpeakerHighIcon/>} onClick={() => setMuted((value) => !value)}>{muted ? '打开声音' : '关闭声音'}</Button>
        <div className={styles.panel}><h2>当前局面</h2><div className={styles.states}>{stateSummary.length === 0 ? <span style={{ color: 'rgb(255 255 255 / 50%)' }}>还没有可见变化</span> : stateSummary.map((item) => <div key={item.id} className={styles.state}><span>{item.name}</span><strong>{item.value}{item.change && <small className="change">{item.change}</small>}</strong></div>)}</div></div>
        {objectives.length > 0 && <div className={styles.panel}><h2>任务进度</h2><div className={styles.taskList}>{objectives.map((item) => <article key={item.id} className={styles.task} data-status={item.status}><div className="taskTop"><span className="taskName">{item.name}</span><strong className="taskStatus">{{ offered: '等待你决定', active: '进行中', completed: '已完成', failed: '已失败', rejected: '已拒绝', locked: '尚未解锁' }[item.status]}</strong></div>{item.issuer && <small className="issuer">来自 {item.issuer}</small>}{item.description && item.status !== 'locked' && <p className="description">{item.description}</p>}{item.status === 'active' && item.promiseText && <p className="description">你的承诺：{item.promiseText}</p>}{item.status === 'active' && item.deadlineTurn && <p className="description">截止：第 {item.deadlineTurn} 回合结束前</p>}</article>)}</div></div>}
        <div className={styles.panel}><h2>你可以自由行动</h2><span style={{ color: 'rgb(255 255 255 / 58%)', lineHeight: 1.6, fontSize: 13 }}>不需要从固定选项里选择。你可以谈判、调查、隐瞒、求助或尝试任何符合当前处境的做法。</span></div>
      </aside>
    </div>
  </div>
}

export default function GamePlayerPage() {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const { projectId, releaseId, sessionId } = useParams()
  const [startKind, setStartKind] = useState<'beginning' | 'turn' | 'event'>('beginning')
  const [startTurn, setStartTurn] = useState(1)
  const [startEventId, setStartEventId] = useState<string>()
  const draftProjectId = releaseId ? undefined : projectId
  const projectQuery = useGameProject(draftProjectId)
  const publishedQuery = usePublishedGame(releaseId)
  const sessionQuery = usePlayerGameSession(projectId, sessionId, releaseId)
  const create = useCreatePlayerGameSession()
  const reset = useResetPlayerGameSession()
  const project = projectQuery.data?.project
  const game = releaseId ? publishedQuery.data?.game : project
  const source: PlayerGameSource | undefined = releaseId ? { source: 'release', releaseId } : project ? { source: 'draft', projectRevision: project.revision } : undefined

  const start = async () => {
    if (!projectId || !source) return
    const session = await create.mutateAsync({ projectId, source, startAt: startKind === 'beginning' ? { kind: 'beginning' } : startKind === 'turn' ? { kind: 'turn', turn: startTurn } : { kind: 'event', eventId: startEventId } })
    const sourcePath = source.source === 'draft' ? 'draft' : `releases/${source.releaseId}`
    void navigate(`/play/games/${projectId}/${sourcePath}/sessions/${session.id}`, { replace: true })
  }

  const restart = async () => {
    if (!projectId || !sessionId) return
    try {
      const session = await reset.mutateAsync({ projectId, sessionId, releaseId })
      const sourcePath = releaseId ? `releases/${releaseId}` : 'draft'
      void navigate(`/play/games/${projectId}/${sourcePath}/sessions/${session.id}`, { replace: true })
    } catch (error) { message.error(parseApiError(error)) }
  }

  if (projectQuery.isLoading || publishedQuery.isLoading || (sessionId && sessionQuery.isLoading)) return <div className={styles.shell}><div className={styles.start}><Skeleton active style={{ width: 560 }}/></div></div>
  if (projectQuery.isError || publishedQuery.isError || (sessionId && sessionQuery.isError)) return <div className={styles.shell}><Result className={styles.result} status="error" title="暂时无法打开这局游戏" subTitle="请检查网络后重试。已经完成的进度不会丢失。" extra={<PrimaryButton onClick={() => window.location.reload()}>重试</PrimaryButton>}/></div>
  if (!game || !projectId) return <div className={styles.shell}><Result className={styles.result} status="404" title="没有找到这个游戏" extra={<Button onClick={() => void navigate('/games')}>返回游戏项目</Button>}/></div>

  return <main className={styles.shell}>
    <header className={styles.header}><button type="button" className={styles.back} onClick={() => releaseId ? void navigate('/') : void navigate(`/games/${projectId}/release`)}><ArrowLeftIcon/>退出试玩</button><div className={styles.gameName}><strong>{game.name}</strong><Tag color={releaseId ? 'green' : 'orange'}>{releaseId ? `正式版本${publishedQuery.data?.version ? ` · v${publishedQuery.data.version}` : ''}` : `草稿试玩 · 第 ${project?.revision ?? '-'} 版`}</Tag></div><div>{sessionId && <Button style={{ marginRight: 8 }} loading={reset.isPending} onClick={() => void restart()}>重新开始</Button>}{!releaseId && <Button type="text" style={{ color: 'rgb(255 255 255 / 70%)' }} icon={<SignOutIcon/>} onClick={() => void navigate(`/games/${projectId}/release`)}>返回编辑</Button>}</div></header>
    {sessionId && sessionQuery.data ? <SessionView session={sessionQuery.data} releaseId={releaseId} metrics={projectQuery.data?.draft.metrics ?? []}/> : <div className={styles.start}><div className={styles.startCard}><span className="draft">{releaseId ? '正式游戏' : '策划试玩 · 不影响已发布版本'}</span><h1>{game.name}</h1><p>{game.description}</p><div className={styles.startOptions}><label>从哪里开始</label><Segmented block value={startKind} options={[{label:'从头开始',value:'beginning'},...(!releaseId?[{label:'指定回合',value:'turn'},{label:'指定事件',value:'event'}]:[])]} onChange={(value)=>setStartKind(value as 'beginning'|'turn'|'event')}/>{startKind==='turn'&&<InputNumber aria-label="开始回合" min={1} max={999} value={startTurn} onChange={(value)=>setStartTurn(value??1)} addonBefore="第" addonAfter="回合"/>}{startKind==='event'&&<Select aria-label="选择开始事件" value={startEventId} placeholder="选择一个事件" options={(projectQuery.data?.draft.events??[]).map((event)=>({value:event.id,label:event.name}))} onChange={setStartEventId}/>}<span style={{color:'rgb(255 255 255 / 50%)',fontSize:12}}>{startKind==='beginning'?'载入初始局面。':startKind==='turn'?'用于快速复测中后段流程。':'应用所选事件的影响后开始试玩。'}</span></div><PrimaryButton size="large" icon={<PlayIcon weight="fill"/>} loading={create.isPending} disabled={startKind==='event'&&!startEventId} onClick={() => void start()}>新建试玩</PrimaryButton>{create.isError && <Alert style={{ marginTop: 16, textAlign: 'left' }} type="error" showIcon title="暂时无法开始" description={parseApiError(create.error)}/>}</div></div>}
  </main>
}
