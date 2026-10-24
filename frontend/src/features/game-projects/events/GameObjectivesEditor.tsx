import { useMemo, useState } from 'react'
import { App, Empty, Form, Input, InputNumber, Modal, Select, Switch, Tag } from 'antd'
import { PencilSimpleIcon, PlusIcon, TrashIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import type { GameProjectCharacter, GameProjectMetric, GameProjectObjective } from '@/api/gameProjects'
import PrimaryButton, { usePrimaryButtonStyle } from '@/components/PrimaryButton'
import { tokens as t } from '@/styles/tokens'

interface Props {
  objectives: GameProjectObjective[]
  metrics: GameProjectMetric[]
  characters: GameProjectCharacter[]
  onChange: (objectives: GameProjectObjective[]) => void
  onSave?: (objectives: GameProjectObjective[]) => Promise<void> | void
  saving?: boolean
}

type ObjectiveFormValue = Omit<GameProjectObjective, 'id'>

const comparisonLabels: Record<string, string> = { gt: '大于', gte: '大于或等于', lt: '小于', lte: '小于或等于', eq: '等于', ne: '不等于' }

const useStyles = createStyles(({ css }) => ({
  intro: css`margin:18px 0 10px;color:${t.textSecondary};line-height:1.65;`,
  toolbar: css`display:flex;align-items:center;justify-content:space-between;gap:14px;margin:14px 0;@media(max-width:620px){align-items:stretch;flex-direction:column;button{width:100%;}}`,
  count: css`color:${t.textSecondary};font-size:${t.textSm};`,
  list: css`display:grid;gap:12px;`,
  card: css`display:grid;grid-template-columns:minmax(0,1fr) auto;gap:14px;padding:18px;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);@media(max-width:620px){grid-template-columns:1fr;}`,
  cardTop: css`display:flex;align-items:center;gap:8px;flex-wrap:wrap;`,
  name: css`margin:0;color:${t.text};font-size:${t.textLg};`,
  description: css`margin:9px 0;color:${t.textSecondary};line-height:1.6;`,
  details: css`display:flex;gap:7px;flex-wrap:wrap;`,
  detail: css`padding:5px 9px;border-radius:999px;color:${t.textSecondary};font-size:12px;background:var(--secondary);`,
  actions: css`display:flex;align-items:flex-start;gap:5px;button{display:grid;place-items:center;width:34px;height:34px;border:1px solid var(--border);border-radius:9px;color:${t.textSecondary};background:var(--card);cursor:pointer;&:hover{color:var(--primary);border-color:var(--primary);}}@media(max-width:620px){justify-content:flex-end;}`,
  empty: css`padding:34px 18px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
  grid: css`display:grid;grid-template-columns:1fr 1fr;gap:0 14px;@media(max-width:680px){grid-template-columns:1fr;}`,
  full: css`grid-column:1/-1;`,
  repeatRow: css`display:grid;grid-template-columns:minmax(0,1fr) 170px 130px auto;gap:8px;align-items:start;margin-bottom:8px;@media(max-width:680px){grid-template-columns:1fr 1fr;.wide{grid-column:1/-1;}}`,
  remove: css`height:32px;border:0;color:var(--error);background:transparent;cursor:pointer;`,
  section: css`grid-column:1/-1;margin:8px 0 12px;padding-top:14px;border-top:1px solid var(--border);strong{display:block;color:${t.text};}p{margin:4px 0 0;color:${t.textTertiary};font-size:12px;line-height:1.55;}`,
  switchItem: css`display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px;border:1px solid var(--border);border-radius:10px;span{color:${t.textSecondary};font-size:${t.textSm};}`,
  saveRow: css`display:flex;justify-content:flex-end;margin-top:18px;`,
}))

function newObjectiveId() {
  return `objective-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`
}

function defaultObjective(): ObjectiveFormValue {
  return { name: '', description: '', kind: 'main', issuerCharacterId: undefined, startsActive: false, dependsOn: [], conditionMode: 'all', conditions: [], failureConditionMode: 'any', failureConditions: [], deadlineTurns: 0, completedEffects: [], failedEffects: [], promiseText: '' }
}

export default function GameObjectivesEditor({ objectives, metrics, characters, onChange, onSave, saving }: Props) {
  const { styles } = useStyles()
  const primary = usePrimaryButtonStyle()
  const { message } = App.useApp()
  const [form] = Form.useForm<ObjectiveFormValue>()
  const [editingId, setEditingId] = useState<string>()
  const [open, setOpen] = useState(false)
  const metricOptions = metrics.map((item) => ({ label: item.name, value: item.id }))
  const characterOptions = characters.filter((item) => item.characterType !== 'observer').map((item) => ({ label: `${item.displayName}（${item.role}）`, value: item.id }))
  const objectiveNames = useMemo(() => Object.fromEntries(objectives.map((item) => [item.id, item.name])), [objectives])
  const characterNames = useMemo(() => Object.fromEntries(characters.map((item) => [item.id, item.displayName])), [characters])

  const openEditor = (objective?: GameProjectObjective) => {
    setEditingId(objective?.id)
    form.resetFields()
    form.setFieldsValue(objective ? {
      ...defaultObjective(),
      ...objective,
      dependsOn: objective.dependsOn,
      conditions: objective.conditions,
      failureConditions: objective.failureConditions ?? [],
    } : defaultObjective())
    setOpen(true)
  }

  const saveObjective = async () => {
    const fields = await form.validateFields()
    const objective: GameProjectObjective = {
      ...fields,
      id: editingId ?? newObjectiveId(),
      dependsOn: fields.dependsOn,
      conditions: fields.conditions,
      failureConditions: fields.failureConditions ?? [],
      deadlineTurns: Math.max(0, fields.deadlineTurns ?? 0),
    }
    const next = editingId ? objectives.map((item) => item.id === editingId ? objective : item) : [...objectives, objective]
    onChange(next)
    setOpen(false)
    message.success(editingId ? '任务已更新' : '任务已添加')
  }

  const removeObjective = (objective: GameProjectObjective) => {
    Modal.confirm({
      title: `删除“${objective.name}”？`,
      content: '以它为前置条件的任务会同时解除这条关联。',
      okText: '删除', cancelText: '取消', okButtonProps: { danger: true },
      onOk: () => { onChange(objectives.filter((item) => item.id !== objective.id).map((item) => ({ ...item, dependsOn: item.dependsOn.filter((id) => id !== objective.id) }))) },
    })
  }

  return <>
    <p className={styles.intro}>任务告诉玩家眼下要解决什么。任务是否完成或失败只看你设置的世界状态，人物不能在对话里擅自改结果。</p>
    <div className={styles.toolbar}><span className={styles.count}>已设置 {objectives.length} 个任务</span><PrimaryButton icon={<PlusIcon/>} onClick={() => { openEditor() }}>新建任务</PrimaryButton></div>
    {objectives.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有玩家任务"><PrimaryButton onClick={() => { openEditor() }}>创建第一个任务</PrimaryButton></Empty></div> : <div className={styles.list}>{objectives.map((objective) => <article className={styles.card} key={objective.id}>
      <div><div className={styles.cardTop}><h3 className={styles.name}>{objective.name}</h3><Tag color={objective.kind === 'main' ? 'orange' : 'default'}>{objective.kind === 'main' ? '主线任务' : '支线任务'}</Tag>{objective.startsActive && <Tag color="green">开局可见</Tag>}</div>
        <p className={styles.description}>{objective.description ?? '还没有填写任务说明'}</p>
        <div className={styles.details}>{objective.issuerCharacterId && <span className={styles.detail}>由 {characterNames[objective.issuerCharacterId] ?? '指定人物'} 交付</span>}<span className={styles.detail}>{objective.dependsOn.length ? `完成 ${objective.dependsOn.map((id) => objectiveNames[id] ?? '前置任务').join('、')} 后解锁` : objective.startsActive ? '开局自动开始' : '等待人物交付'}</span><span className={styles.detail}>{objective.conditions.length} 条完成条件</span>{(objective.deadlineTurns ?? 0) > 0 && <span className={styles.detail}>{objective.deadlineTurns} 回合内完成</span>}{(objective.failureConditions?.length ?? 0) > 0 && <span className={styles.detail}>{objective.failureConditions?.length} 条失败条件</span>}</div>
      </div>
      <div className={styles.actions}><button type="button" aria-label={`编辑${objective.name}`} onClick={() => { openEditor(objective) }}><PencilSimpleIcon/></button><button type="button" aria-label={`删除${objective.name}`} onClick={() => { removeObjective(objective) }}><TrashIcon/></button></div>
    </article>)}</div>}
    {onSave && <div className={styles.saveRow}><PrimaryButton size="large" loading={saving} onClick={() => void onSave(objectives)}>保存任务设置</PrimaryButton></div>}

    <Modal open={open} width={860} title={editingId ? '编辑任务' : '新建任务'} okText="保存任务" cancelText="取消" okButtonProps={{ className: primary.root }} onCancel={() => { setOpen(false) }} onOk={() => void saveObjective()} destroyOnHidden>
      <Form form={form} layout="vertical" requiredMark={false} initialValues={defaultObjective()}>
        <div className={styles.grid}>
          <Form.Item name="name" label="任务名称" rules={[{ required: true, message: '请填写任务名称' }]}><Input maxLength={80} placeholder="例如：封存关键证据"/></Form.Item>
          <Form.Item name="kind" label="任务类型"><Select options={[{ value: 'main', label: '主线任务' }, { value: 'optional', label: '支线任务' }]}/></Form.Item>
          <Form.Item className={styles.full} name="description" label="告诉玩家要做什么" rules={[{ required: true, message: '请填写任务说明' }]}><Input.TextArea rows={3} maxLength={500} showCount placeholder="用玩家看得懂的话说明目标，不要写内部规则。"/></Form.Item>
          <Form.Item name="issuerCharacterId" label="由谁交付" extra="选择人物后，该人物可以在对话中正式交付任务。"><Select allowClear placeholder="没有指定人物" options={characterOptions}/></Form.Item>
          <div className={styles.switchItem}><span>开局直接显示</span><Form.Item name="startsActive" valuePropName="checked" noStyle><Switch/></Form.Item></div>
          <Form.Item className={styles.full} name="dependsOn" label="先完成哪些任务" extra="这些任务全部完成后，本任务才会解锁。"><Select mode="multiple" allowClear placeholder="没有前置任务" options={objectives.filter((item) => item.id !== editingId).map((item) => ({ value: item.id, label: item.name }))}/></Form.Item>

          <div className={styles.section}><strong>怎样算完成</strong><p>只使用真实世界状态判断，避免人物口头说完成就算完成。</p></div>
          <Form.Item className={styles.full} name="conditionMode" label="条件组合"><Select options={[{ value: 'all', label: '以下条件全部满足' }, { value: 'any', label: '以下条件满足任意一条' }]}/></Form.Item>
          <ConditionList formName="conditions" addLabel="添加完成条件" metrics={metricOptions} styles={styles}/>

          <div className={styles.section}><strong>期限与失败</strong><p>这些设置可以留空。填写后，系统会在到期或命中失败条件时结束任务。</p></div>
          <Form.Item name="deadlineTurns" label="需要在几回合内完成" extra="填 0 表示不限时。"><InputNumber min={0} max={999} style={{ width: '100%' }}/></Form.Item>
          <Form.Item name="failureConditionMode" label="失败条件组合"><Select options={[{ value: 'any', label: '任意一条满足即失败' }, { value: 'all', label: '全部满足才失败' }]}/></Form.Item>
          <ConditionList formName="failureConditions" addLabel="添加失败条件" metrics={metricOptions} styles={styles}/>
          <Form.Item className={styles.full} name="promiseText" label="玩家接受后记住的承诺" extra="可选。用于复盘完成、违约及其后果。"><Input.TextArea rows={2} maxLength={300} placeholder="例如：我承诺在董事会召开前封存全部原始凭证。"/></Form.Item>
        </div>
      </Form>
    </Modal>
  </>
}

function ConditionList({ formName, addLabel, metrics, styles }: { formName: 'conditions' | 'failureConditions'; addLabel: string; metrics: { label: string; value: string }[]; styles: Record<string, string> }) {
  return <Form.List name={formName}>{(fields, { add, remove }) => <div className={styles.full}>{fields.map(({ key, name }) => <div className={styles.repeatRow} key={key}>
    <Form.Item name={[name, 'target']} rules={[{ required: true, message: '选择世界状态' }]}><Select className="wide" placeholder="选择世界状态" options={metrics}/></Form.Item>
    <Form.Item name={[name, 'operator']} rules={[{ required: true, message: '选择比较方式' }]}><Select placeholder="如何比较" options={Object.entries(comparisonLabels).map(([value, label]) => ({ value, label }))}/></Form.Item>
    <Form.Item name={[name, 'value']} rules={[{ required: true, message: '填写数值' }]}><InputNumber style={{ width: '100%' }} placeholder="数值"/></Form.Item>
    <button className={styles.remove} type="button" aria-label={`删除${addLabel}`} onClick={() => { remove(name) }}><TrashIcon size={18}/></button>
  </div>)}<PrimaryButton ghost size="small" onClick={() => { add({ operator: 'gte', value: 0 }) }}>+{addLabel}</PrimaryButton></div>}</Form.List>
}
