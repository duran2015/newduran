import { useMemo, useState } from 'react'
import { Alert, App, Empty, Form, Input, InputNumber, Modal, Select, Switch, Tag } from 'antd'
import { FlaskIcon, PencilSimpleIcon, PlusIcon, TrashIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import type { GameEventCategory, GameProjectAction, GameProjectCharacter, GameProjectEvent, GameProjectMetric, PreviewGameEventsResult } from '@/api/gameProjects'
import PrimaryButton from '@/components/PrimaryButton'
import { usePreviewGameEvents } from '@/queries/useGameProjects'
import { tokens as t } from '@/styles/tokens'
import GameSessionWorkspace from '../session/GameSessionWorkspace'

export interface GameEventsValue {
  events: GameProjectEvent[]
  metrics: GameProjectMetric[]
  actions: GameProjectAction[]
  characters: GameProjectCharacter[]
  scenes?: { id: string; name: string }[]
}

interface Props {
  projectId: string
  value: GameEventsValue
  onChange: (events: GameProjectEvent[]) => void
  onSave?: (events: GameProjectEvent[]) => Promise<void> | void
  saving?: boolean
}

type EventFormValue = Omit<GameProjectEvent, 'id'>

const categoryLabels: Record<GameEventCategory, string> = {
  system: '系统事件',
  route: '路线事件',
  personal: '人物私事',
  environment: '环境事件',
}

const comparisonLabels: Record<string, string> = { gt: '大于', gte: '大于或等于', lt: '小于', lte: '小于或等于', eq: '等于', ne: '不等于' }
const operationLabels: Record<string, string> = { increment: '增加', decrement: '减少', set: '设为' }

const useStyles = createStyles(({ css }) => ({
  intro: css`margin:20px 0 16px;`,
  toolbar: css`display:flex;align-items:center;justify-content:space-between;gap:14px;margin:18px 0 14px;@media(max-width:620px){align-items:stretch;flex-direction:column;button{width:100%;}}`,
  count: css`color:${t.textSecondary};font-size:${t.textSm};`,
  list: css`display:grid;gap:12px;`,
  card: css`display:grid;grid-template-columns:minmax(0,1fr) auto;gap:14px;padding:18px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);@media(max-width:620px){grid-template-columns:1fr;}`,
  cardTop: css`display:flex;align-items:center;gap:9px;flex-wrap:wrap;`,
  name: css`margin:0;color:${t.text};font-size:${t.textLg};`,
  content: css`margin:10px 0;color:${t.textSecondary};line-height:1.65;white-space:pre-wrap;`,
  details: css`display:flex;gap:7px;flex-wrap:wrap;`,
  detail: css`padding:5px 9px;border-radius:999px;color:${t.textSecondary};font-size:12px;background:var(--secondary);`,
  actions: css`display:flex;align-items:flex-start;gap:5px;button{display:grid;place-items:center;width:34px;height:34px;border:1px solid var(--border);border-radius:9px;color:${t.textSecondary};background:var(--card);cursor:pointer;&:hover{color:var(--primary);border-color:var(--primary);}}@media(max-width:620px){justify-content:flex-end;}`,
  empty: css`padding:34px 18px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
  formIntro: css`margin:-4px 0 18px;color:${t.textTertiary};line-height:1.55;`,
  grid: css`display:grid;grid-template-columns:1fr 1fr;gap:0 14px;@media(max-width:680px){grid-template-columns:1fr;}`,
  full: css`grid-column:1/-1;`,
  repeatRow: css`display:grid;grid-template-columns:minmax(0,1fr) 150px 130px auto;gap:8px;align-items:start;margin-bottom:8px;@media(max-width:680px){grid-template-columns:1fr 1fr;.wide{grid-column:1/-1;}}`,
  remove: css`height:32px;border:0;color:var(--error);background:transparent;cursor:pointer;`,
  sectionHint: css`margin:-8px 0 12px;color:${t.textTertiary};font-size:12px;`,
  switches: css`display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px 14px;grid-column:1/-1;margin-bottom:14px;@media(max-width:680px){grid-template-columns:1fr;}`,
  switchItem: css`display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;border:1px solid var(--border);border-radius:10px;span{color:${t.textSecondary};font-size:${t.textSm};}`,
  saveRow: css`display:flex;justify-content:flex-end;margin-top:18px;`,
  trialGrid: css`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0 12px;@media(max-width:560px){grid-template-columns:1fr;}`,
  result: css`display:grid;gap:10px;margin-top:16px;`,
  resultCard: css`padding:14px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);`,
  resultTitle: css`display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:8px;font-weight:650;`,
  resultText: css`margin:4px 0;color:${t.textSecondary};line-height:1.6;`,
}))

function newEventId() {
  return `event-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`
}

function defaultEvent(): EventFormValue {
  return { name: '', description: '', category: 'system', displayContent: '', conditionMode: 'all', conditions: [], participantIds: [], effects: [], advancesTime: false, timeAdvance: 0, priority: 0, once: true, cooldownTurns: 0, exclusiveGroup: '', nextEventIds: [], endsGame: false }
}

export default function GameEventsEditor({ projectId, value, onChange, onSave, saving }: Props) {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const [form] = Form.useForm<EventFormValue>()
  const [editingId, setEditingId] = useState<string>()
  const [editorOpen, setEditorOpen] = useState(false)
  const [trialEvent, setTrialEvent] = useState<GameProjectEvent>()
  const [trialState, setTrialState] = useState<Record<string, number>>({})
  const [trialTurn, setTrialTurn] = useState(1)
  const [trialResult, setTrialResult] = useState<PreviewGameEventsResult>()
  const previewMutation = usePreviewGameEvents(projectId)

  const metricOptions = value.metrics.map((item) => ({ label: item.name, value: item.id }))
  const characterOptions = value.characters.map((item) => ({ label: `${item.displayName}（${item.role}）`, value: item.id }))
  const eventOptions = value.events.filter((item) => item.id !== editingId).map((item) => ({ label: item.name, value: item.id }))
  const metricNames = useMemo(() => Object.fromEntries(value.metrics.map((item) => [item.id, item.name])), [value.metrics])

  const openEditor = (event?: GameProjectEvent) => {
    setEditingId(event?.id)
    form.resetFields()
    form.setFieldsValue(event ? { ...event } : defaultEvent())
    setEditorOpen(true)
  }

  const saveEvent = async () => {
    const fields = await form.validateFields()
    const event: GameProjectEvent = { ...fields, id: editingId ?? newEventId(), timeAdvance: fields.advancesTime ? Math.max(1, fields.timeAdvance || 1) : 0 }
    const next = editingId ? value.events.map((item) => item.id === editingId ? event : item) : [...value.events, event]
    onChange(next)
    setEditorOpen(false)
    message.success(editingId ? '事件已更新' : '事件已添加')
  }

  const removeEvent = (event: GameProjectEvent) => {
    Modal.confirm({
      title: `删除“${event.name}”？`,
      content: '已将它设为后续的其他事件，会同时移除这条后续关联。',
      okText: '删除', cancelText: '取消', okButtonProps: { danger: true },
      onOk: () => onChange(value.events.filter((item) => item.id !== event.id).map((item) => ({ ...item, nextEventIds: item.nextEventIds.filter((id) => id !== event.id) }))),
    })
  }

  const openTrial = (event: GameProjectEvent) => {
    setTrialEvent(event)
    setTrialState(Object.fromEntries(value.metrics.map((item) => [item.id, item.initial])))
    setTrialTurn(1)
    setTrialResult(undefined)
  }

  const runTrial = async () => {
    if (!trialEvent) return
    try {
      await onSave?.(value.events)
      const result = await previewMutation.mutateAsync({ state: trialState, currentTurn: trialTurn, candidateEventIds: [trialEvent.id] })
      setTrialResult(result)
    } catch {
      message.error('试跑失败，请先保存事件并检查填写内容')
    }
  }

  return <>
    <Alert className={styles.intro} showIcon type="info" title="事件决定故事何时发生" description="当世界状态满足条件时，系统会选出应该发生的事件，邀请相关人物参与，并结算对世界的影响。"/>
    <div className={styles.toolbar}><span className={styles.count}>已设置 {value.events.length} 个事件</span><PrimaryButton icon={<PlusIcon/>} onClick={() => openEditor()}>新建事件</PrimaryButton></div>
    {value.events.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有故事事件"><PrimaryButton onClick={() => openEditor()}>创建第一个事件</PrimaryButton></Empty></div> : <div className={styles.list}>{value.events.map((event) => <article className={styles.card} key={event.id}>
      <div><div className={styles.cardTop}><h3 className={styles.name}>{event.name}</h3><Tag>{categoryLabels[event.category] ?? '其他事件'}</Tag>{event.once && <Tag color="green">仅发生一次</Tag>}{event.endsGame && <Tag color="red">会结束游戏</Tag>}</div>
        <p className={styles.content}>{event.displayContent || event.description || '未填写玩家会看到的内容'}</p>
        <div className={styles.details}><span className={styles.detail}>{event.conditions.length ? `${event.conditionMode === 'all' ? '全部' : '任一'} ${event.conditions.length} 个条件` : '无条件候选'}</span><span className={styles.detail}>{event.participantIds.length} 个参与人物</span><span className={styles.detail}>{event.effects.length} 项数值影响</span>{event.advancesTime && <span className={styles.detail}>推进 {event.timeAdvance} 回合</span>}<span className={styles.detail}>优先级 {event.priority}</span></div>
      </div>
      <div className={styles.actions}><button type="button" aria-label={`试跑${event.name}`} onClick={() => openTrial(event)}><FlaskIcon/></button><button type="button" aria-label={`编辑${event.name}`} onClick={() => openEditor(event)}><PencilSimpleIcon/></button><button type="button" aria-label={`删除${event.name}`} onClick={() => removeEvent(event)}><TrashIcon/></button></div>
    </article>)}</div>}
    {onSave && <div className={styles.saveRow}><PrimaryButton size="large" loading={saving} onClick={() => void onSave(value.events)}>保存事件设置</PrimaryButton></div>}
    <GameSessionWorkspace projectId={projectId} metrics={value.metrics} characters={value.characters}/>

    <Modal open={editorOpen} width={860} title={editingId ? '编辑事件' : '新建事件'} okText="保存事件" cancelText="取消" onCancel={() => setEditorOpen(false)} onOk={() => void saveEvent()} destroyOnHidden>
      <p className={styles.formIntro}>说清楚什么时候发生、谁会参与，以及故事会如何变化。</p>
      <Form form={form} layout="vertical" requiredMark={false} initialValues={defaultEvent()}>
        <div className={styles.grid}>
          <Form.Item name="name" label="事件名称" rules={[{ required: true, message: '请填写事件名称' }]}><Input maxLength={80} placeholder="例如：隐藏债务被曝光"/></Form.Item>
          <Form.Item name="category" label="事件类型" rules={[{ required: true }]}><Select options={Object.entries(categoryLabels).map(([valueKey, label]) => ({ value: valueKey, label }))}/></Form.Item>
          <Form.Item className={styles.full} name="displayContent" label="发生时玩家会看到什么" rules={[{ required: true, message: '请填写展示内容' }]}><Input.TextArea rows={4} maxLength={1200} showCount placeholder="用故事语言描述发生了什么，不要填技术说明。"/></Form.Item>
          <Form.Item className={styles.full} name="description" label="策划备注"><Input.TextArea rows={2} maxLength={400} placeholder="可选：记录这个事件的设计意图，玩家不会看到。"/></Form.Item>
          <Form.Item className={styles.full} name="sceneId" label="发生后切换到哪个场景" extra="留空保持当前场景。切换后使用该场景的人物立绘、背景音乐和入场音效。"><Select allowClear placeholder="保持当前场景" options={(value.scenes ?? []).map((scene) => ({ value: scene.id, label: scene.name }))}/></Form.Item>
          <Form.Item className={styles.full} name="endReason" label="结局说明" extra="仅在事件设置为结束游戏时显示给玩家。"><Input.TextArea rows={2} placeholder="描述这次故事如何结束，以及玩家的选择造成了什么结果。"/></Form.Item>
          <Form.Item className={styles.full} name="conditionMode" label="何时发生"><Select options={[{ value: 'all', label: '以下条件全部满足' }, { value: 'any', label: '以下条件满足任意一条' }]}/></Form.Item>
          <Form.List name="conditions">{(fields, { add, remove }) => <div className={styles.full}>{fields.map(({ key, name }) => <div className={styles.repeatRow} key={key}>
            <Form.Item name={[name, 'target']} rules={[{ required: true, message: '选择世界状态' }]}><Select className="wide" placeholder="选择世界状态" options={metricOptions}/></Form.Item>
            <Form.Item name={[name, 'operator']} rules={[{ required: true, message: '选择比较方式' }]}><Select placeholder="如何比较" options={Object.entries(comparisonLabels).map(([valueKey, label]) => ({ value: valueKey, label }))}/></Form.Item>
            <Form.Item name={[name, 'value']} rules={[{ required: true, message: '填写数值' }]}><InputNumber style={{ width: '100%' }} placeholder="数值"/></Form.Item><button className={styles.remove} type="button" aria-label="删除发生条件" onClick={() => remove(name)}><TrashIcon size={18}/></button>
          </div>)}<PrimaryButton ghost size="small" onClick={() => add({ operator: 'gte', value: 0 })}>+添加发生条件</PrimaryButton></div>}</Form.List>
          <Form.Item className={styles.full} name="participantIds" label="哪些人物会参与"><Select mode="multiple" options={characterOptions} placeholder={characterOptions.length ? '选择参与这件事的人物' : '请先在上一步添加人物'}/></Form.Item>
          <div className={styles.full}><strong>发生后，世界会怎么变</strong><p className={styles.sectionHint}>这些变化由系统结算，不由人物自行编造。</p></div>
          <Form.List name="effects">{(fields, { add, remove }) => <div className={styles.full}>{fields.map(({ key, name }) => <div className={styles.repeatRow} key={key}>
            <Form.Item name={[name, 'target']} rules={[{ required: true, message: '选择世界状态' }]}><Select className="wide" placeholder="选择世界状态" options={metricOptions}/></Form.Item>
            <Form.Item name={[name, 'operation']} rules={[{ required: true, message: '选择变化方式' }]}><Select placeholder="变化方式" options={Object.entries(operationLabels).map(([valueKey, label]) => ({ value: valueKey, label }))}/></Form.Item>
            <Form.Item name={[name, 'value']} rules={[{ required: true, message: '填写数值' }]}><InputNumber style={{ width: '100%' }} placeholder="数值"/></Form.Item><button className={styles.remove} type="button" aria-label="删除数值影响" onClick={() => remove(name)}><TrashIcon size={18}/></button>
          </div>)}<PrimaryButton ghost size="small" onClick={() => add({ operation: 'increment', value: 0 })}>+添加数值影响</PrimaryButton></div>}</Form.List>
          <Form.Item name="priority" label="优先级" tooltip="同一回合多个事件都能发生时，数字越大越先处理。"><InputNumber min={-100} max={100} style={{ width: '100%' }}/></Form.Item>
          <Form.Item name="cooldownTurns" label="再次发生需间隔几回合"><InputNumber min={0} max={999} style={{ width: '100%' }}/></Form.Item>
          <Form.Item name="exclusiveGroup" label="互斥组" tooltip="同组事件在一个回合只会选中一个。"><Input maxLength={60} placeholder="可选，例如：首日危机"/></Form.Item>
          <Form.Item name="nextEventIds" label="接下来可能发生"><Select mode="multiple" options={eventOptions} placeholder="可选：选择后续事件"/></Form.Item>
          <div className={styles.switches}>
            <div className={styles.switchItem}><span>仅发生一次</span><Form.Item name="once" valuePropName="checked" noStyle><Switch/></Form.Item></div>
            <div className={styles.switchItem}><span>发生后推进回合</span><Form.Item name="advancesTime" valuePropName="checked" noStyle><Switch/></Form.Item></div>
            <div className={styles.switchItem}><span>发生后结束游戏</span><Form.Item name="endsGame" valuePropName="checked" noStyle><Switch/></Form.Item></div>
          </div>
          <Form.Item noStyle shouldUpdate={(previous, current) => previous.advancesTime !== current.advancesTime}>{({ getFieldValue }) => getFieldValue('advancesTime') ? <Form.Item className={styles.full} name="timeAdvance" label="推进多少回合" rules={[{ required: true }]}><InputNumber min={1} max={999} style={{ width: '100%' }}/></Form.Item> : null}</Form.Item>
        </div>
      </Form>
    </Modal>

    <Modal open={Boolean(trialEvent)} width={720} title={trialEvent ? `试跑：${trialEvent.name}` : '试跑事件'} okText="开始试跑" cancelText="关闭" confirmLoading={previewMutation.isPending || saving} onOk={() => void runTrial()} onCancel={() => setTrialEvent(undefined)}>
      <p className={styles.formIntro}>设定当前回合和世界状态，由真实游戏规则判断它会不会发生。试跑不会改动游戏内容。</p>
      <div className={styles.trialGrid}><label><span>当前回合</span><InputNumber min={0} value={trialTurn} onChange={(next) => setTrialTurn(next ?? 0)} style={{ width: '100%', marginTop: 7 }}/></label>{value.metrics.map((metric) => <label key={metric.id}><span>{metric.name}{metric.displayUnit ? `（${metric.displayUnit}）` : ''}</span><InputNumber min={metric.minimum} max={metric.maximum} value={trialState[metric.id]} onChange={(next) => setTrialState((current) => ({ ...current, [metric.id]: next ?? 0 }))} style={{ width: '100%', marginTop: 7 }}/></label>)}</div>
      {trialResult && <div className={styles.result}><Alert showIcon type={(trialResult.items[0]?.eligible ?? false) ? 'success' : 'warning'} title={trialResult.summary}/>{trialResult.items.map((item) => <div className={styles.resultCard} key={item.eventId}><div className={styles.resultTitle}><span>{item.name}</span><Tag color={item.eligible ? 'green' : 'orange'}>{item.eligible ? '条件已满足' : '现在不会发生'}</Tag>{item.selected && <Tag color="blue">本回合选中</Tag>}</div>{item.reasons.map((reason) => <p className={styles.resultText} key={reason}>{reason}</p>)}{item.stateDiff.length > 0 && <p className={styles.resultText}><strong>数值影响：</strong>{item.stateDiff.map((diff) => `${metricNames[diff.metricId] ?? '某项状态'} ${String(diff.before)} → ${String(diff.after)}`).join('；')}</p>}{item.timeAdvance > 0 && <p className={styles.resultText}>将推进 {item.timeAdvance} 回合</p>}{item.endsGame && <p className={styles.resultText}>这个事件会结束游戏</p>}</div>)}</div>}
    </Modal>
  </>
}
