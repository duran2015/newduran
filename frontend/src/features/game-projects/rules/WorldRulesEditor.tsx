import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Card, Divider, Drawer, Empty, Form, Input, InputNumber, Popconfirm, Select, Space, Tag, message } from 'antd'
import { CalculatorIcon, GaugeIcon, LightningIcon, PencilSimpleIcon, PlusIcon, TrashIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import type { GameProjectAction, GameProjectEvent, GameProjectMetric, GameStateChangeMethod, GameStateComparison } from '@/api/gameProjects'
import PrimaryButton from '@/components/PrimaryButton'
import { usePreviewGameAction } from '@/queries/useGameProjects'
import { tokens as t } from '@/styles/tokens'

export interface WorldRulesValue {
  metrics: GameProjectMetric[]
  actions: GameProjectAction[]
}

interface WorldRulesEditorProps {
  projectId: string
  value: WorldRulesValue
  events?: GameProjectEvent[]
  onChange: (value: WorldRulesValue) => void
  onSave?: (value: WorldRulesValue) => Promise<void> | void
  saving?: boolean
}

type MetricFormValue = Omit<GameProjectMetric, 'id'>
type ActionFormValue = Omit<GameProjectAction, 'id'>

const comparisonOptions: { value: GameStateComparison; label: string }[] = [
  { value: 'gt', label: '大于' }, { value: 'gte', label: '不低于' },
  { value: 'lt', label: '小于' }, { value: 'lte', label: '不高于' }, { value: 'eq', label: '等于' }, { value: 'ne', label: '不等于' },
]
const effectOptions: { value: GameStateChangeMethod; label: string }[] = [
  { value: 'increment', label: '增加' }, { value: 'decrement', label: '减少' }, { value: 'set', label: '变为' },
]

const makeId = (prefix: string) => `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`

const useStyles = createStyles(({ css }) => ({
  root: css`display:grid;gap:22px;`,
  sectionHead: css`display:flex;justify-content:space-between;align-items:flex-start;gap:14px;@media(max-width:620px){align-items:stretch;flex-direction:column;}`,
  sectionTitle: css`display:flex;align-items:center;gap:9px;margin:0;color:${t.text};font-size:18px;`,
  hint: css`margin:5px 0 0;color:${t.textTertiary};line-height:1.55;`,
  grid: css`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;@media(max-width:720px){grid-template-columns:1fr;}`,
  card: css`height:100%;border-color:var(--border);background:var(--background);.ant-card-body{height:100%;display:flex;flex-direction:column;}`,
  cardTop: css`display:flex;justify-content:space-between;align-items:flex-start;gap:10px;`,
  cardTitle: css`margin:0;color:${t.text};font-size:16px;`,
  description: css`min-height:42px;margin:7px 0 14px;color:${t.textTertiary};font-size:${t.textSm};line-height:1.5;`,
  facts: css`display:flex;flex-wrap:wrap;gap:7px;margin-top:auto;`,
  empty: css`padding:26px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
  trial: css`padding:18px;border:1px solid color-mix(in srgb,var(--primary) 22%,var(--border));border-radius:${t.radiusLg}px;background:var(--primary-subtle);`,
  trialInputs: css`display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px;margin:16px 0;@media(max-width:800px){grid-template-columns:repeat(2,minmax(0,1fr));}@media(max-width:520px){grid-template-columns:1fr;}`,
  inputLabel: css`display:grid;gap:5px;color:${t.textSecondary};font-size:12px;.ant-input-number{width:100%;}`,
  resultGrid: css`display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-top:12px;@media(max-width:620px){grid-template-columns:1fr;}`,
  resultItem: css`padding:10px 12px;border:1px solid var(--border);border-radius:${t.radiusSm}px;background:var(--card);strong{display:block;color:${t.text};}span{color:${t.textMuted};font-size:12px;}`,
  listRow: css`display:grid;grid-template-columns:minmax(0,1fr) 150px 110px 42px;gap:8px;align-items:center;margin-bottom:8px;@media(max-width:680px){grid-template-columns:minmax(0,1fr) 110px 38px;.comparison{grid-column:1/-1;}}`,
  footer: css`display:flex;justify-content:flex-end;gap:10px;padding-top:4px;@media(max-width:520px){button{flex:1;}}`,
}))

export default function WorldRulesEditor({ projectId, value, events = [], onChange, onSave, saving }: WorldRulesEditorProps) {
  const { styles } = useStyles()
  const [metricForm] = Form.useForm<MetricFormValue>()
  const [actionForm] = Form.useForm<ActionFormValue>()
  const [metricOpen, setMetricOpen] = useState(false)
  const [actionOpen, setActionOpen] = useState(false)
  const [editingMetric, setEditingMetric] = useState<string>()
  const [editingAction, setEditingAction] = useState<string>()
  const [trialActionId, setTrialActionId] = useState<string>()
  const [trialState, setTrialState] = useState<Record<string, number>>(() => Object.fromEntries(value.metrics.map((metric) => [metric.id, metric.initial])))
  const previewMutation = usePreviewGameAction(projectId)
  const trialResult = previewMutation.data
  const metricOptions = useMemo(() => value.metrics.map((metric) => ({ value: metric.id, label: metric.name })), [value.metrics])
  const eventOptions = useMemo(() => events.map((event) => ({ value: event.id, label: event.endsGame ? `${event.name}（结局）` : event.name })), [events])
  const eventNames = useMemo(() => Object.fromEntries(events.map((event) => [event.id, event.name])), [events])

  useEffect(() => {
    setTrialState((current) => Object.fromEntries(value.metrics.map((metric) => [metric.id, current[metric.id] ?? metric.initial])))
  }, [value.metrics])

  const openMetric = (metric?: GameProjectMetric) => {
    setEditingMetric(metric?.id)
    metricForm.setFieldsValue(metric ?? { name: '', description: '', type: 'number', initial: 50, minimum: 0, maximum: 100, displayUnit: '' })
    setMetricOpen(true)
  }
  const saveMetric = async () => {
    const fields = await metricForm.validateFields()
    const metric = { id: editingMetric ?? makeId('state'), ...fields }
    onChange({ ...value, metrics: editingMetric ? value.metrics.map((item) => item.id === editingMetric ? metric : item) : [...value.metrics, metric] })
    setMetricOpen(false)
    message.success(editingMetric ? '状态项已更新' : '状态项已添加')
  }
  const removeMetric = (id: string) => {
    onChange({ metrics: value.metrics.filter((metric) => metric.id !== id), actions: value.actions.filter((action) => !(action.conditions ?? []).some((item) => item.target === id) && !(action.effects ?? []).some((item) => item.target === id)) })
    message.success('状态项已删除，引用它的行动也已移除')
  }
  const openAction = (action?: GameProjectAction) => {
    setEditingAction(action?.id)
    actionForm.setFieldsValue(action ?? { name: '', description: '', conditionMode: 'all', conditions: [], effects: [{ target: value.metrics[0]?.id, operation: 'increment', value: 1 }], timeCost: 0 })
    setActionOpen(true)
  }
  const saveAction = async () => {
    const fields = await actionForm.validateFields()
    const action = { id: editingAction ?? makeId('action'), ...fields }
    onChange({ ...value, actions: editingAction ? value.actions.map((item) => item.id === editingAction ? action : item) : [...value.actions, action] })
    setActionOpen(false)
    message.success(editingAction ? '行动已更新' : '行动已添加')
  }
  const runTrial = async () => {
    const action = value.actions.find((item) => item.id === trialActionId)
    if (!action) return message.warning('请先选择一个行动')
    try { await previewMutation.mutateAsync({ actionId: action.id, state: trialState, requestKey: `preview-${Date.now()}` }) }
    catch { message.error('试算失败，请先保存当前规则后再试') }
  }

  return <div className={styles.root}>
    <section>
      <div className={styles.sectionHead}><div><h3 className={styles.sectionTitle}><GaugeIcon size={20}/>世界状态</h3><p className={styles.hint}>这些数值共同描述游戏当前局面。设置合理范围，系统会阻止数值超出边界。</p></div><Button icon={<PlusIcon/>} onClick={() => openMetric()}>添加状态</Button></div>
      <div style={{height:12}}/>
      {value.metrics.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有世界状态"><PrimaryButton onClick={() => openMetric()}>添加财富、舆情等状态</PrimaryButton></Empty></div> : <div className={styles.grid}>{value.metrics.map((metric) => <Card key={metric.id} className={styles.card}><div className={styles.cardTop}><div><h4 className={styles.cardTitle}>{metric.name}</h4><Tag color="blue">初始 {metric.initial}{metric.displayUnit}</Tag></div><Space><Button aria-label={`编辑${metric.name}`} type="text" icon={<PencilSimpleIcon/>} onClick={() => openMetric(metric)}/><Popconfirm title="删除这个状态项？" description="引用它的行动也会一并移除。" okText="删除" cancelText="取消" onConfirm={() => removeMetric(metric.id)}><Button aria-label={`删除${metric.name}`} danger type="text" icon={<TrashIcon/>}/></Popconfirm></Space></div><p className={styles.description}>{metric.description || '暂未填写说明'}</p><div className={styles.facts}><Tag>最低 {metric.minimum ?? '不限'}</Tag><Tag>最高 {metric.maximum ?? '不限'}</Tag>{metric.displayUnit && <Tag>{metric.displayUnit}</Tag>}</div></Card>)}</div>}
    </section>
    <Divider/>
    <section>
      <div className={styles.sectionHead}><div><h3 className={styles.sectionTitle}><LightningIcon size={20}/>玩家与人物可以采取的行动</h3><p className={styles.hint}>每个行动可以有执行条件，并同时改变多个世界状态。</p></div><Button icon={<PlusIcon/>} disabled={value.metrics.length === 0} onClick={() => openAction()}>添加行动</Button></div>
      <div style={{height:12}}/>
      {value.actions.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={value.metrics.length ? '还没有设置行动' : '请先添加至少一个世界状态'}>{value.metrics.length > 0 && <PrimaryButton onClick={() => openAction()}>添加第一个行动</PrimaryButton>}</Empty></div> : <div className={styles.grid}>{value.actions.map((action) => <Card key={action.id} className={styles.card}><div className={styles.cardTop}><div><h4 className={styles.cardTitle}>{action.name}</h4>{action.timeCost > 0 && <Tag color="orange">推进 {action.timeCost} 个时间单位</Tag>}</div><Space><Button aria-label={`编辑${action.name}`} type="text" icon={<PencilSimpleIcon/>} onClick={() => openAction(action)}/><Popconfirm title="删除这个行动？" okText="删除" cancelText="取消" onConfirm={() => onChange({ ...value, actions: value.actions.filter((item) => item.id !== action.id) })}><Button aria-label={`删除${action.name}`} danger type="text" icon={<TrashIcon/>}/></Popconfirm></Space></div><p className={styles.description}>{action.description || '暂未填写说明'}</p><div className={styles.facts}><Tag>{(action.conditions ?? []).length ? `${action.conditions.length} 个条件` : '随时可执行'}</Tag><Tag>{(action.effects ?? []).length} 项影响</Tag>{action.emits && <Tag color="purple">接着发生：{eventNames[action.emits] ?? action.emits}</Tag>}</div></Card>)}</div>}
    </section>
    <Divider/>
    <section className={styles.trial}>
      <div className={styles.sectionHead}><div><h3 className={styles.sectionTitle}><CalculatorIcon size={20}/>试算行动结果</h3><p className={styles.hint}>调整当前数值，再选择行动。这里只预演结果，不会改动游戏草稿。</p></div><Button onClick={() => { setTrialState(Object.fromEntries(value.metrics.map((metric) => [metric.id, metric.initial]))); previewMutation.reset() }}>恢复初始值</Button></div>
      <div className={styles.trialInputs}>{value.metrics.map((metric) => <label className={styles.inputLabel} key={metric.id}><span>{metric.name}</span><InputNumber aria-label={`当前${metric.name}`} min={metric.minimum} max={metric.maximum} value={trialState[metric.id]} onChange={(number) => setTrialState((state) => ({ ...state, [metric.id]: number ?? metric.initial }))}/></label>)}</div>
      <Space.Compact block><Select aria-label="选择要试算的行动" placeholder="选择一个行动" options={value.actions.map((action) => ({value:action.id,label:action.name}))} value={trialActionId} onChange={(id) => { setTrialActionId(id); previewMutation.reset() }} style={{flex:1}}/><PrimaryButton loading={previewMutation.isPending} onClick={() => void runTrial()}>开始试算</PrimaryButton></Space.Compact>
      {trialResult && <div style={{marginTop:14}}><Alert showIcon type={trialResult.allowed ? 'success' : 'warning'} title={trialResult.allowed ? '可以执行这个行动' : '目前不能执行'} description={trialResult.reasons.join('；')}/>{trialResult.allowed && <div className={styles.resultGrid}>{trialResult.stateDiff.length ? trialResult.stateDiff.map((change) => <div className={styles.resultItem} key={change.metricId}><strong>{value.metrics.find((metric) => metric.id === change.metricId)?.name ?? '状态变化'}</strong><span>{String(change.before)} → {String(change.after)}</span></div>) : <div className={styles.resultItem}><strong>世界状态不变</strong><span>这个行动没有设置数值影响</span></div>}</div>}</div>}
    </section>
    {onSave && <div className={styles.footer}><PrimaryButton size="large" loading={saving} onClick={() => void onSave(value)}>保存规则与世界状态</PrimaryButton></div>}

    <Drawer title={editingMetric ? '编辑世界状态' : '添加世界状态'} width={520} open={metricOpen} onClose={() => setMetricOpen(false)} extra={<PrimaryButton onClick={() => void saveMetric()}>完成</PrimaryButton>}>
      <Form form={metricForm} layout="vertical" requiredMark={false}><Form.Item name="name" label="名称" rules={[{required:true,message:'请填写名称'}]}><Input maxLength={40} placeholder="例如：监管压力"/></Form.Item><Form.Item name="description" label="说明"><Input.TextArea rows={3} maxLength={200} placeholder="这个数值代表什么？玩家在哪里能看到它？"/></Form.Item><Form.Item name="type" hidden><Input/></Form.Item><Form.Item name="displayUnit" label="显示单位"><Input maxLength={12} placeholder="例如：亿元、%（可不填）"/></Form.Item><div className={styles.grid}><Form.Item name="initial" label="初始值" rules={[{required:true,message:'请填写初始值'}]}><InputNumber style={{width:'100%'}}/></Form.Item><Form.Item name="minimum" label="最低值" rules={[{required:true,message:'请填写最低值'}]}><InputNumber style={{width:'100%'}}/></Form.Item><Form.Item name="maximum" label="最高值" rules={[{required:true,message:'请填写最高值'},({getFieldValue})=>({validator(_, number){return number > getFieldValue('minimum') ? Promise.resolve() : Promise.reject(new Error('最高值必须高于最低值'))}})]}><InputNumber style={{width:'100%'}}/></Form.Item></div></Form>
    </Drawer>

    <Drawer title={editingAction ? '编辑行动' : '添加行动'} width={680} open={actionOpen} onClose={() => setActionOpen(false)} extra={<PrimaryButton onClick={() => void saveAction()}>完成</PrimaryButton>}>
      <Form form={actionForm} layout="vertical" requiredMark={false}><Form.Item name="name" label="行动名称" rules={[{required:true,message:'请填写行动名称'}]}><Input maxLength={60} placeholder="例如：主动披露"/></Form.Item><Form.Item name="description" label="行动说明"><Input.TextArea rows={3} maxLength={260} placeholder="玩家采取这个行动时会发生什么？"/></Form.Item><Form.Item name="timeCost" label="消耗多少时间" tooltip="填写 0 表示不推进游戏时间" rules={[{required:true,message:'请填写耗时'}]}><InputNumber min={0} precision={0} addonAfter="个时间单位" style={{width:'100%'}}/></Form.Item><Form.Item name="emits" label="执行后直接推动哪个故事事件" extra="例如“封存证据”后直接进入“财务调查结论”。留空时，系统只按事件自身条件判断。"><Select allowClear showSearch optionFilterProp="label" placeholder={eventOptions.length ? '可选：选择一个故事事件' : '请先在“事件与推进”中添加事件'} options={eventOptions}/></Form.Item><Form.Item name="conditionMode" label="多个条件如何判断"><Select options={[{value:'all',label:'全部满足才能执行'},{value:'any',label:'满足任意一个即可执行'}]}/></Form.Item><Divider titlePlacement="left">执行条件</Divider><Form.List name="conditions">{(fields,{add,remove}) => <>{fields.map(({key,name,...rest}) => <div className={styles.listRow} key={key}><Form.Item {...rest} name={[name,'target']} rules={[{required:true,message:'请选择状态'}]} noStyle><Select placeholder="选择状态" options={metricOptions}/></Form.Item><Form.Item {...rest} name={[name,'operator']} rules={[{required:true,message:'请选择条件'}]} noStyle><Select className="comparison" options={comparisonOptions}/></Form.Item><Form.Item {...rest} name={[name,'value']} rules={[{required:true,message:'请输入数值'}]} noStyle><InputNumber style={{width:'100%'}}/></Form.Item><Button aria-label="删除条件" danger type="text" icon={<TrashIcon/>} onClick={() => remove(name)}/></div>)}<Button type="dashed" block icon={<PlusIcon/>} onClick={() => add({target:value.metrics[0]?.id,operator:'gte',value:0})}>添加条件</Button></>}</Form.List><Divider titlePlacement="left">执行后的影响</Divider><Form.List name="effects" rules={[{validator:async(_,items)=>{if(!items?.length) throw new Error('请至少设置一项影响')}}]}>{(fields,{add,remove},{errors}) => <>{fields.map(({key,name,...rest}) => <div className={styles.listRow} key={key}><Form.Item {...rest} name={[name,'target']} rules={[{required:true,message:'请选择状态'}]} noStyle><Select placeholder="选择状态" options={metricOptions}/></Form.Item><Form.Item {...rest} name={[name,'operation']} rules={[{required:true,message:'请选择变化方式'}]} noStyle><Select className="comparison" options={effectOptions}/></Form.Item><Form.Item {...rest} name={[name,'value']} rules={[{required:true,message:'请输入数值'}]} noStyle><InputNumber min={0} style={{width:'100%'}}/></Form.Item><Button aria-label="删除影响" danger type="text" icon={<TrashIcon/>} onClick={() => remove(name)}/></div>)}<Button type="dashed" block icon={<PlusIcon/>} onClick={() => add({target:value.metrics[0]?.id,operation:'increment',value:1})}>添加一项影响</Button><Form.ErrorList errors={errors}/></>}</Form.List></Form>
    </Drawer>
  </div>
}
