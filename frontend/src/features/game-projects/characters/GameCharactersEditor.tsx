import { useMemo, useState } from 'react'
import { Alert, App, Empty, Form, Input, Modal, Select, Spin, Tag } from 'antd'
import { EyeIcon, PencilSimpleIcon, PlusIcon, RobotIcon, TrashIcon, WarningCircleIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import type { Agent } from '@/api/agents'
import type { AgentRelation } from '@/api/agent-relations'
import { gameProjectsApi, type GameCharacterContextPreview, type GameProjectAction, type GameProjectCharacter, type GameProjectMetric } from '@/api/gameProjects'
import { unwrapResponse } from '@/api/client'
import PrimaryButton from '@/components/PrimaryButton'
import { useAgents } from '@/queries/useAgents'
import { useAgentRelations } from '@/queries/useAgentRelations'
import { tokens as t } from '@/styles/tokens'

export interface GameCharactersValue {
  characters: GameProjectCharacter[]
  metrics: GameProjectMetric[]
  actions: GameProjectAction[]
}

interface Props {
  projectId: string
  value: GameCharactersValue
  onChange: (characters: GameProjectCharacter[]) => void
  onSave?: (characters: GameProjectCharacter[]) => Promise<void> | void
  saving?: boolean
}

type CharacterFormValue = {
  agentId: number
  displayName: string
  role: string
  responsibilities?: string
  characterType: 'director' | 'npc' | 'observer'
  visibleMetricIds: string[]
  executableActionIds: string[]
  advisoryActionIds: string[]
  knownInformationText?: string
  relationIds: number[]
}

const roleTypeLabels: Record<GameProjectCharacter['characterType'], string> = {
  director: '游戏导演',
  npc: '普通人物',
  observer: '观察者',
}

const useStyles = createStyles(({ css }) => ({
  intro: css`margin:20px 0 16px;`,
  toolbar: css`display:flex;align-items:center;justify-content:space-between;gap:14px;margin:18px 0 14px;@media(max-width:620px){align-items:stretch;flex-direction:column;button{width:100%;}}`,
  count: css`color:${t.textSecondary};font-size:${t.textSm};`,
  list: css`display:grid;gap:12px;`,
  card: css`display:grid;grid-template-columns:minmax(0,1fr) auto;gap:14px;padding:17px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);@media(max-width:620px){grid-template-columns:1fr;}`,
  cardTop: css`display:flex;align-items:center;gap:10px;flex-wrap:wrap;`,
  avatar: css`display:grid;place-items:center;width:38px;height:38px;border-radius:12px;color:var(--primary);background:var(--primary-subtle);`,
  name: css`margin:0;color:${t.text};font-size:${t.textLg};`,
  title: css`margin:3px 0 0;color:${t.textTertiary};font-size:${t.textSm};`,
  details: css`display:flex;gap:8px;flex-wrap:wrap;margin-top:13px;`,
  detail: css`padding:5px 9px;border-radius:999px;color:${t.textSecondary};font-size:12px;background:var(--secondary);`,
  warning: css`display:flex;align-items:center;gap:6px;margin-top:12px;color:var(--warning);font-size:${t.textSm};`,
  actions: css`display:flex;align-items:flex-start;gap:5px;button{display:grid;place-items:center;width:34px;height:34px;border:1px solid var(--border);border-radius:9px;color:${t.textSecondary};background:var(--card);cursor:pointer;&:hover{color:var(--primary);border-color:var(--primary);}}@media(max-width:620px){justify-content:flex-end;}`,
  empty: css`padding:34px 18px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
  modalIntro: css`margin:-4px 0 18px;color:${t.textTertiary};line-height:1.55;`,
  formGrid: css`display:grid;grid-template-columns:1fr 1fr;gap:0 14px;@media(max-width:620px){grid-template-columns:1fr;}`,
  full: css`grid-column:1/-1;`,
  preview: css`display:grid;gap:14px;`,
  previewBlock: css`padding:14px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);strong{display:block;margin-bottom:6px;color:${t.text};}p{margin:0;color:${t.textSecondary};line-height:1.6;white-space:pre-wrap;}`,
  saveRow: css`display:flex;justify-content:flex-end;margin-top:18px;`,
}))

function agentDisplayName(agent: Agent) {
  const title = agent.config?.title
  if (title && typeof title === 'object') return title['zh-CN'] || title.zh || title.en || agent.name
  return agent.name
}

const relationTypeLabels: Record<string, string> = {
  reports_to: '汇报关系', peer: '同事协作', advisor: '顾问关系', reviewer: '复核关系',
  oversight: '监督关系', representative: '代表关系', opponent: '对抗关系', external: '外部联系',
}

function relationLabel(relation: AgentRelation) {
  return `${relation.sourceAgentName} → ${relation.targetAgentName} · ${relationTypeLabels[relation.relationType] ?? '已有关系'}`
}

function newCharacterId() {
  return `character-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`
}

export default function GameCharactersEditor({ projectId, value, onChange, onSave, saving }: Props) {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const agentsQuery = useAgents()
  const agents = agentsQuery.data ?? []
  const relationsQuery = useAgentRelations()
  const relations = relationsQuery.data ?? []
  const [form] = Form.useForm<CharacterFormValue>()
  const [editingId, setEditingId] = useState<string>()
  const [modalOpen, setModalOpen] = useState(false)
  const [previewing, setPreviewing] = useState<GameProjectCharacter>()
  const [contextPreview, setContextPreview] = useState<GameCharacterContextPreview>()
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState(false)
  const agentIds = useMemo(() => new Set(agents.map((agent) => agent.id)), [agents])
  const metricOptions = value.metrics.map((metric) => ({ label: metric.name, value: metric.id }))
  const actionOptions = value.actions.map((action) => ({ label: action.name, value: action.id }))

  const openEditor = (character?: GameProjectCharacter) => {
    setEditingId(character?.id)
    form.resetFields()
    form.setFieldsValue(character ? {
      agentId: character.agentId,
      displayName: character.displayName,
      role: character.role,
      responsibilities: character.responsibilities,
      characterType: character.characterType,
      visibleMetricIds: character.visibleMetricIds ?? [],
      executableActionIds: character.executableActionIds ?? [],
      advisoryActionIds: character.advisoryActionIds ?? [],
      knownInformationText: (character.knownInformation ?? []).join('\n'),
      relationIds: character.relationIds ?? [],
    } : { characterType: 'npc', visibleMetricIds: [], executableActionIds: [], advisoryActionIds: [], relationIds: [] })
    setModalOpen(true)
  }

  const saveCharacter = async () => {
    const fields = await form.validateFields()
    const existing = value.characters.find((item) => item.id === editingId)
    const selectedAgent = agents.find((agent) => agent.id === fields.agentId)
    const character: GameProjectCharacter = {
      id: editingId ?? newCharacterId(),
      agentId: fields.agentId,
      displayName: fields.displayName,
      role: fields.role,
      responsibilities: fields.responsibilities,
      characterType: fields.characterType,
      visibleMetricIds: fields.visibleMetricIds ?? [],
      executableActionIds: fields.executableActionIds ?? [],
      advisoryActionIds: fields.advisoryActionIds ?? [],
      knownInformation: (fields.knownInformationText ?? '').split('\n').map((line) => line.trim()).filter(Boolean),
      informationScope: existing?.informationScope ?? '仅限这个人物',
      knowledgeRefs: existing?.knowledgeRefs ?? selectedAgent?.datasets ?? [],
      relationIds: fields.relationIds ?? [],
      permissions: existing?.permissions ?? [],
    }
    const next = editingId ? value.characters.map((item) => item.id === editingId ? character : item) : [...value.characters, character]
    onChange(next)
    setModalOpen(false)
    message.success(editingId ? '人物设置已更新' : '人物已加入游戏')
  }

  const removeCharacter = (character: GameProjectCharacter) => {
    Modal.confirm({
      title: `从游戏中移除“${character.displayName}”？`,
      content: '只会解除这款游戏的引用，不会删除 Agent Hub 中的 Agent。',
      okText: '移除', cancelText: '取消', okButtonProps: { danger: true },
      onOk: () => { onChange(value.characters.filter((item) => item.id !== character.id)) },
    })
  }

  const openContextPreview = async (character: GameProjectCharacter) => {
    setPreviewing(character)
    setContextPreview(undefined)
    setPreviewError(false)
    setPreviewLoading(true)
    try {
      const result = unwrapResponse<GameCharacterContextPreview>(await gameProjectsApi.previewCharacterContext(projectId, character.id))
      setContextPreview(result)
    } catch {
      setPreviewError(true)
    } finally {
      setPreviewLoading(false)
    }
  }

  if (agentsQuery.isLoading) return <Spin tip="正在读取 Agent Hub 中的人物"><div style={{ minHeight: 220 }}/></Spin>

  return <>
    <Alert className={styles.intro} showIcon type="info" title="游戏引用 Agent，不会复制 Agent" description="人格、模型和知识仍由 Agent Hub 统一维护；这里仅设置他在这款游戏中的身份、权限和掌握的信息。"/>
    {agentsQuery.isError && <Alert showIcon type="warning" title="暂时无法读取 Agent Hub" description="已经添加的人物仍会保留。恢复连接后，可继续选择或检查 Agent。"/>}
    <div className={styles.toolbar}><span className={styles.count}>已加入 {value.characters.length} 个人物</span><PrimaryButton icon={<PlusIcon/>} onClick={() => { openEditor() }}>从 Agent Hub 选择人物</PrimaryButton></div>
    {value.characters.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有为这款游戏添加人物"><PrimaryButton onClick={() => { openEditor() }}>选择第一个 Agent</PrimaryButton></Empty></div> : <div className={styles.list}>{value.characters.map((character) => {
      const missing = !agentIds.has(character.agentId)
      const visibleNames = (character.visibleMetricIds ?? []).map((id) => value.metrics.find((metric) => metric.id === id)?.name).filter(Boolean)
      return <article className={styles.card} key={character.id}>
        <div><div className={styles.cardTop}><span className={styles.avatar}><RobotIcon size={20}/></span><div><h3 className={styles.name}>{character.displayName}</h3><p className={styles.title}>{character.role}</p></div><Tag>{roleTypeLabels[character.characterType]}</Tag></div>
          <div className={styles.details}><span className={styles.detail}>可见 {visibleNames.length} 项状态</span><span className={styles.detail}>可执行 {(character.executableActionIds ?? []).length} 个行动</span><span className={styles.detail}>可建议 {(character.advisoryActionIds ?? []).length} 个行动</span><span className={styles.detail}>引用 {(character.relationIds ?? []).length} 条现有关系</span></div>
          {missing && <div className={styles.warning}><WarningCircleIcon size={17}/>Agent Hub 中已找不到这个 Agent，请重新选择后再试玩。</div>}
        </div>
        <div className={styles.actions}><button type="button" aria-label={`预览${character.displayName}收到的信息`} onClick={() => { void openContextPreview(character) }}><EyeIcon/></button><button type="button" aria-label={`编辑${character.displayName}`} onClick={() => { openEditor(character) }}><PencilSimpleIcon/></button><button type="button" aria-label={`移除${character.displayName}`} onClick={() => { removeCharacter(character) }}><TrashIcon/></button></div>
      </article>
    })}</div>}
    {onSave && <div className={styles.saveRow}><PrimaryButton size="large" loading={saving} onClick={() => { void onSave(value.characters) }}>保存人物设置</PrimaryButton></div>}

    <Modal open={modalOpen} width={760} title={editingId ? '编辑游戏人物' : '从 Agent Hub 添加人物'} okText="保存人物" cancelText="取消" onCancel={() => { setModalOpen(false) }} onOk={() => { void saveCharacter() }} destroyOnHidden>
      <p className={styles.modalIntro}>先选择现有 Agent，再说明他在这款游戏里是谁、能做什么、知道什么。</p>
      <Form form={form} layout="vertical" requiredMark={false}>
        <div className={styles.formGrid}>
          <Form.Item className={styles.full} name="agentId" label="选择 Agent" rules={[{ required: true, message: '请选择一个 Agent' }]}><Select showSearch optionFilterProp="label" placeholder="搜索 Agent" options={agents.map((agent) => ({ label: agentDisplayName(agent), value: agent.id, disabled: value.characters.some((item) => item.id !== editingId && item.agentId === agent.id) }))} onChange={(agentId) => { if (!form.getFieldValue('displayName')) form.setFieldValue('displayName', agentDisplayName(agents.find((agent) => agent.id === agentId)!)); form.setFieldValue('relationIds', []) }}/></Form.Item>
          <Form.Item name="displayName" label="游戏内姓名" rules={[{ required: true, message: '请填写游戏内姓名' }]}><Input maxLength={60} placeholder="例如：莫云岑"/></Form.Item>
          <Form.Item name="role" label="身份或职位" rules={[{ required: true, message: '请填写身份或职位' }]}><Input maxLength={80} placeholder="例如：首席财务官"/></Form.Item>
          <Form.Item className={styles.full} name="responsibilities" label="在故事中的职责"><Input.TextArea rows={2} maxLength={300} placeholder="他负责什么，会在什么事情上参与判断？"/></Form.Item>
          <Form.Item name="characterType" label="人物类型" rules={[{ required: true }]}><Select options={Object.entries(roleTypeLabels).map(([valueKey, label]) => ({ value: valueKey, label, disabled: valueKey === 'director' && value.characters.some((item) => item.id !== editingId && item.characterType === 'director') }))}/></Form.Item>
          <Form.Item name="visibleMetricIds" label="可以看到的世界状态"><Select mode="multiple" options={metricOptions} placeholder={metricOptions.length ? '选择他能看到的状态' : '请先设置世界状态'}/></Form.Item>
          <Form.Item name="executableActionIds" label="可以直接执行的行动"><Select mode="multiple" options={actionOptions} placeholder={actionOptions.length ? '选择允许执行的行动' : '请先设置行动'}/></Form.Item>
          <Form.Item name="advisoryActionIds" label="可以提出建议的行动"><Select mode="multiple" options={actionOptions} placeholder={actionOptions.length ? '选择允许建议的行动' : '请先设置行动'}/></Form.Item>
          <Form.Item className={styles.full} name="knownInformationText" label="掌握的信息"><Input.TextArea rows={4} maxLength={1600} showCount placeholder="每行填写一条只有这个人物知道的事实、线索或秘密。"/></Form.Item>
          <Form.Item noStyle shouldUpdate={(previous, current) => previous.agentId !== current.agentId}>{({ getFieldValue }) => {
            const agentId = getFieldValue('agentId') as number | undefined
            const addedAgentIds = new Set(value.characters.filter((item) => item.id !== editingId).map((item) => item.agentId))
            const options = relations.filter((relation) => agentId && (relation.sourceAgentId === agentId || relation.targetAgentId === agentId)).filter((relation) => addedAgentIds.has(relation.sourceAgentId === agentId ? relation.targetAgentId : relation.sourceAgentId)).map((relation) => ({ value: relation.id, label: relationLabel(relation) }))
            return <Form.Item className={styles.full} name="relationIds" label="沿用 Agent Hub 中的关系"><Select mode="multiple" options={options} placeholder={options.length ? '选择这款游戏要使用的现有关系' : '双方加入游戏后，这里会显示可用关系'} notFoundContent="当前没有双方都已加入游戏的现有关系"/></Form.Item>
          }}</Form.Item>
        </div>
      </Form>
    </Modal>

    <Modal open={Boolean(previewing)} title={previewing ? `${previewing.displayName}进入游戏时会知道什么` : '人物信息预览'} footer={null} onCancel={() => { setPreviewing(undefined) }}>
      {previewing && <div className={styles.preview}>
        {previewLoading && <Spin tip="正在生成真实信息范围"><div style={{ minHeight: 160 }}/></Spin>}
        {previewError && <Alert showIcon type="warning" title="请先保存人物设置" description="保存后，系统才能按正式权限生成这个人物真正会收到的信息。"/>}
        {contextPreview && <>
          <Alert showIcon type={contextPreview.agentAvailable ? 'info' : 'warning'} title={contextPreview.availableSummary} description="这里展示的是系统实际会交给这个人物的信息，不是页面临时拼出的说明。"/>
          <div className={styles.previewBlock}><strong>身份与职责</strong><p>{contextPreview.identity}{contextPreview.responsibilities ? `\n${contextPreview.responsibilities}` : ''}</p></div>
          <div className={styles.previewBlock}><strong>可以看到的状态</strong><p>{(contextPreview.visibleWorldState ?? []).join('、') || '未授权查看任何世界状态'}</p></div>
          <div className={styles.previewBlock}><strong>可以采取的行动</strong><p>{(contextPreview.canExecute ?? []).join('、') || '不能直接执行行动'}</p></div>
          <div className={styles.previewBlock}><strong>可以提出的建议</strong><p>{(contextPreview.canSuggest ?? []).join('、') || '没有可建议的行动'}</p></div>
          <div className={styles.previewBlock}><strong>掌握的信息</strong><p>{(contextPreview.knownInformation ?? []).join('\n') || '没有额外信息'}{contextPreview.knowledgeSummary ? `\n${contextPreview.knowledgeSummary}` : ''}</p></div>
          <div className={styles.previewBlock}><strong>与其他人物的关系</strong><p>{(contextPreview.relationships ?? []).join('\n') || '未使用现有关系'}</p></div>
          <div className={styles.previewBlock}><strong>不能做的事</strong><p>{(contextPreview.limits ?? []).join('\n') || '没有额外限制'}</p></div>
        </>}
      </div>}
    </Modal>
  </>
}
