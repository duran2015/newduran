import { useEffect, useMemo, useState } from 'react'
import { Alert, App, Empty, Form, Input, Progress, Result, Select, Skeleton, Tabs, Tag } from 'antd'
import { ArrowLeftIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import { useNavigate, useParams } from 'react-router'
import PrimaryButton from '@/components/PrimaryButton'
import { getGameProjectProgress, type GameProjectCreateInput } from '@/api/gameProjects'
import { useDeleteGameProjectAsset, useGameProject, useGameProjectAssets, useReplaceGameProjectAsset, useUpdateGameProject, useUploadGameProjectAsset } from '@/queries/useGameProjects'
import { tokens as t } from '@/styles/tokens'
import WorldRulesEditor, { type WorldRulesValue } from './rules/WorldRulesEditor'
import GameCharactersEditor from './characters/GameCharactersEditor'
import type { GameProjectCharacter, GameProjectEvent, GameProjectObjective, GameProjectScene } from '@/api/gameProjects'
import GameEventsEditor from './events/GameEventsEditor'
import GameObjectivesEditor from './events/GameObjectivesEditor'
import GameScenesEditor from './scenes/GameScenesEditor'
import GameReleaseWorkspace from './release/GameReleaseWorkspace'
import { GAME_EDITOR_STEPS } from '@/lib/nav'

const steps = GAME_EDITOR_STEPS

type StepKey = typeof steps[number]['key']

const useStyles = createStyles(({ css }) => ({
  page: css`max-width:1400px;margin:0 auto;`,
  top: css`display:flex;justify-content:space-between;align-items:flex-start;gap:20px;margin-bottom:20px;@media(max-width:680px){flex-direction:column;}`,
  back: css`display:inline-flex;align-items:center;gap:6px;margin-bottom:12px;padding:0;border:0;color:${t.textSecondary};background:transparent;cursor:pointer;`,
  title: css`display:flex;align-items:center;gap:10px;margin:0;color:${t.text};font-size:${t.text2xl};letter-spacing:-.03em;`,
  summary: css`max-width:70ch;margin:7px 0 0;color:${t.textTertiary};line-height:1.6;`,
  layout: css`min-height:690px;overflow:hidden;border:1px solid var(--border);border-radius:${t.radiusLg}px;background:var(--card);box-shadow:${t.elevation1};`,
  mobileSteps: css`display:none;padding:14px 18px;border-bottom:1px solid var(--border);background:color-mix(in srgb,var(--background) 72%,var(--card));@media(max-width:768px){display:block;}`,
  content: css`min-width:0;padding:28px;@media(max-width:640px){padding:18px;}`,
  sectionTop: css`display:flex;justify-content:space-between;align-items:flex-start;gap:16px;padding-bottom:20px;border-bottom:1px solid var(--border);`,
  sectionTitle: css`margin:0;color:${t.text};font-size:${t.textXl};`,
  sectionText: css`max-width:70ch;margin:6px 0 0;color:${t.textTertiary};line-height:1.6;`,
  cards: css`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:20px;@media(max-width:680px){grid-template-columns:1fr;}`,
  card: css`padding:17px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);strong{display:block;color:${t.text};}p{margin:6px 0 0;color:${t.textTertiary};font-size:${t.textSm};line-height:1.55;}`,
  metric: css`font-size:${t.text2xl};font-weight:700;color:var(--primary);`,
  empty: css`margin-top:24px;padding:36px 20px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
  releaseFlow: css`display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin:20px 0;@media(max-width:720px){grid-template-columns:1fr;}`,
  releaseStep: css`padding:15px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);strong{display:block;margin-bottom:5px;color:${t.text};}span{color:${t.textMuted};font-size:${t.textSm};line-height:1.5;}`,
  overviewForm: css`max-width:760px;margin-top:22px;`,
  formActions: css`display:flex;align-items:center;gap:12px;margin-top:8px;@media(max-width:520px){align-items:stretch;flex-direction:column;button{width:100%;}}`,
  progress: css`width:210px;padding:12px 14px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--card);@media(max-width:680px){width:100%;}`,
  progressLabel: css`display:flex;justify-content:space-between;gap:12px;margin-bottom:6px;color:${t.textSecondary};font-size:12px;`,
  loading: css`padding:28px;border:1px solid var(--border);border-radius:${t.radiusLg}px;background:var(--card);`,
}))

export default function GameProjectEditorPage() {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const { projectId, step } = useParams()
  const projectQuery = useGameProject(projectId)
  const assetsQuery = useGameProjectAssets(projectId)
  const updateMutation = useUpdateGameProject(projectId)
  const uploadAssetMutation = useUploadGameProjectAsset(projectId)
  const deleteAssetMutation = useDeleteGameProjectAsset(projectId)
  const replaceAssetMutation = useReplaceGameProjectAsset(projectId)
  const active: StepKey = steps.some((item) => item.key === step) ? step as StepKey : 'overview'
  const [worldRules, setWorldRules] = useState<WorldRulesValue>({ metrics: [], actions: [] })
  const [characters, setCharacters] = useState<GameProjectCharacter[]>([])
  const [events, setEvents] = useState<GameProjectEvent[]>([])
  const [objectives, setObjectives] = useState<GameProjectObjective[]>([])
  const [scenes, setScenes] = useState<GameProjectScene[]>([])
  const [form] = Form.useForm<GameProjectCreateInput>()
  const project = projectQuery.data?.project
  const progress = useMemo(() => projectQuery.data ? getGameProjectProgress(projectQuery.data) : undefined, [projectQuery.data])
  const selectedStep = steps.find((item) => item.key === active) ?? steps[0]

  useEffect(() => {
    if (!project) return
    form.setFieldsValue({ name: project.name, description: project.description, genre: project.genre, playGoal: project.playGoal })
  }, [form, project])

  useEffect(() => {
    if (!projectQuery.data) return
    const draft = projectQuery.data.draft
    setWorldRules({
      metrics: draft.metrics ?? [],
      actions: (draft.actions ?? []).map((action) => ({
        ...action,
        conditions: action.conditions ?? [],
        effects: action.effects ?? [],
      })),
    })
    setCharacters(draft.npcs ?? [])
    setEvents((draft.events ?? []).map((event) => ({
      ...event,
      conditions: event.conditions ?? [],
      effects: event.effects ?? [],
      participantIds: event.participantIds ?? event.participantCharacterIds ?? [],
      nextEventIds: event.nextEventIds ?? [],
    })))
    setObjectives((draft.objectives ?? []).map((objective) => ({
      ...objective,
      kind: objective.kind || 'main',
      dependsOn: objective.dependsOn ?? [],
      conditions: objective.conditions ?? [],
      failureConditions: objective.failureConditions ?? [],
    })))
    setScenes(draft.scenes ?? [])
  }, [projectQuery.data])

  if (projectQuery.isLoading) return <div className={styles.loading}><Skeleton active paragraph={{ rows: 8 }}/></div>
  if (projectQuery.isError) return <Result status="error" title="暂时无法打开这个游戏" subTitle="请检查网络后重试，已保存的内容不会丢失。" extra={<PrimaryButton onClick={() => { void projectQuery.refetch() }}>重试</PrimaryButton>}/>
  if (!project || !projectQuery.data) return <Empty description="没有找到这个游戏项目" />

  const saveOverview = async () => {
    const values = await form.validateFields()
    try {
      await updateMutation.mutateAsync({ revision: project.revision, ...values })
      message.success('游戏资料已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请稍后重试')
    }
  }

  const saveWorldRules = async (rules: WorldRulesValue) => {
    try {
      await updateMutation.mutateAsync({
        revision: project.revision,
        name: project.name,
        description: project.description,
        genre: project.genre,
        playGoal: project.playGoal,
        draft: { ...projectQuery.data.draft, metrics: rules.metrics, actions: rules.actions },
      })
      message.success('规则与世界状态已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请稍后重试')
      throw error
    }
  }

  const saveCharacters = async (nextCharacters: GameProjectCharacter[]) => {
    try {
      await updateMutation.mutateAsync({
        revision: project.revision,
        name: project.name,
        description: project.description,
        genre: project.genre,
        playGoal: project.playGoal,
        draft: { ...projectQuery.data.draft, npcs: nextCharacters },
      })
      message.success('人物设置已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请检查人物设置后重试')
      throw error
    }
  }

  const saveEvents = async (nextEvents: GameProjectEvent[]) => {
    try {
      await updateMutation.mutateAsync({
        revision: project.revision,
        name: project.name,
        description: project.description,
        genre: project.genre,
        playGoal: project.playGoal,
        draft: { ...projectQuery.data.draft, events: nextEvents },
      })
      message.success('事件与推进设置已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请检查事件设置后重试')
      throw error
    }
  }

  const saveObjectives = async (nextObjectives: GameProjectObjective[]) => {
    try {
      await updateMutation.mutateAsync({
        revision: project.revision,
        name: project.name,
        description: project.description,
        genre: project.genre,
        playGoal: project.playGoal,
        draft: { ...projectQuery.data.draft, objectives: nextObjectives },
      })
      message.success('任务设置已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请检查任务设置后重试')
      throw error
    }
  }

  const saveScenes = async (nextScenes: GameProjectScene[]) => {
    try {
      await updateMutation.mutateAsync({
        revision: project.revision,
        name: project.name,
        description: project.description,
        genre: project.genre,
        playGoal: project.playGoal,
        draft: { ...projectQuery.data.draft, scenes: nextScenes },
      })
      message.success('场景与视听已保存')
    } catch (error) {
      const conflict = typeof error === 'object' && error !== null && 'response' in error && (error as { response?: { status?: number } }).response?.status === 409
      message.error(conflict ? '这个项目刚被其他人修改，请刷新后再保存' : '保存失败，请检查场景设置后重试')
      throw error
    }
  }

  const renderContent = () => {
    if (active === 'overview') return <Form className={styles.overviewForm} form={form} layout="vertical" requiredMark={false}>
      <Form.Item name="name" label="游戏名称" rules={[{required:true,message:'请填写游戏名称'}]}><Input maxLength={80} showCount placeholder="例如：Speeding"/></Form.Item>
      <Form.Item name="description" label="游戏简介" rules={[{required:true,message:'请说明这是一个怎样的游戏'}]}><Input.TextArea rows={4} maxLength={600} showCount placeholder="玩家将面对什么局面？"/></Form.Item>
      <Form.Item name="genre" label="题材" rules={[{required:true,message:'请填写题材'}]}><Input maxLength={80} placeholder="例如：财富与权力策略叙事"/></Form.Item>
      <Form.Item name="playGoal" label="玩法目标" rules={[{required:true,message:'请说明玩家要达成什么目标'}]}><Input.TextArea rows={3} maxLength={400} showCount placeholder="玩家需要做什么，以及想要达到什么结果？"/></Form.Item>
      <div className={styles.formActions}><PrimaryButton size="large" loading={updateMutation.isPending} onClick={() => { void saveOverview() }}>保存游戏资料</PrimaryButton><span>保存后，可以继续配置规则、人物和故事。</span></div>
    </Form>
    if (active === 'world') return <WorldRulesEditor projectId={project.id} value={worldRules} events={events} onChange={setWorldRules} onSave={saveWorldRules} saving={updateMutation.isPending}/>
    if (active === 'characters') return <GameCharactersEditor projectId={project.id} value={{ characters, metrics: worldRules.metrics, actions: worldRules.actions }} onChange={setCharacters} onSave={saveCharacters} saving={updateMutation.isPending}/>
    if (active === 'events') return <Tabs defaultActiveKey="events" items={[
      { key: 'events', label: '故事事件', children: <GameEventsEditor projectId={project.id} value={{ events, metrics: worldRules.metrics, actions: worldRules.actions, characters, scenes }} onChange={setEvents} onSave={saveEvents} saving={updateMutation.isPending}/> },
      { key: 'objectives', label: `玩家任务${objectives.length ? `（${objectives.length}）` : ''}`, children: <GameObjectivesEditor objectives={objectives} metrics={worldRules.metrics} characters={characters} onChange={setObjectives} onSave={saveObjectives} saving={updateMutation.isPending}/> },
    ]}/>
    if (active === 'scenes') return <>
      {assetsQuery.isError && <Alert showIcon type="warning" title="暂时无法读取资源库" description="已编辑的场景仍会保留。请检查网络后重试。"/>}
      <GameScenesEditor
        assets={assetsQuery.data ?? []}
        scenes={scenes}
        characters={characters}
        onChange={setScenes}
        onSave={saveScenes}
        saving={updateMutation.isPending}
        onUpload={async (file, kind, onProgress) => { await uploadAssetMutation.mutateAsync({ file, name: file.name.replace(/\.[^.]+$/, ''), kind, onProgress }) }}
        onReplaceAsset={async (asset, file, onProgress) => { await replaceAssetMutation.mutateAsync({ assetId: asset.id, file, name: asset.name, onProgress }) }}
        onDeleteAsset={async (asset) => { await deleteAssetMutation.mutateAsync(asset.id) }}
      />
    </>
    if (active === 'release') return <GameReleaseWorkspace detail={projectQuery.data}/>
    return null
  }

  return <main className={styles.page}>
    <header className={styles.top}>
      <div><button type="button" className={styles.back} onClick={() => { void navigate('/games') }}><ArrowLeftIcon size={16}/>返回游戏项目</button><h1 className={styles.title}>{project.name}<Tag>{projectQuery.data.releases.length > 0 ? '已发布' : '制作中'}</Tag></h1><p className={styles.summary}>{project.description}</p></div>
      <div className={styles.progress}><div className={styles.progressLabel}><span>制作进度</span><strong>{progress?.count ?? 0}/6 步</strong></div><Progress percent={progress?.percent ?? 0} showInfo={false} size="small"/></div>
    </header>
    <div className={styles.layout}>
      <div className={styles.mobileSteps}><Select aria-label="选择游戏制作步骤" value={active} style={{ width: '100%' }} options={steps.map((item, index) => ({ value: item.key, label: `${index + 1}. ${item.label}${progress?.completed[index] ? ' · 已完成' : ''}` }))} onChange={(next: StepKey) => { void navigate(`/games/${project.id}/${next}`) }}/></div>
      <section className={styles.content}>
        <div className={styles.sectionTop}><div><h2 className={styles.sectionTitle}>{selectedStep.label}</h2><p className={styles.sectionText}>{selectedStep.hint}</p></div>{!progress?.completed[steps.findIndex((item) => item.key === active)] && active !== 'overview' && <Tag>尚未完成</Tag>}</div>
        {renderContent()}
      </section>
    </div>
  </main>
}
