import { useState } from 'react'
import { Alert, Empty, Form, Input, Modal, Progress, Skeleton, Tag } from 'antd'
import { ArrowRightIcon, GameControllerIcon, PlusIcon } from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import { useNavigate } from 'react-router'
import PrimaryButton, { usePrimaryButtonStyle } from '@/components/PrimaryButton'
import { useCreateGameProject, useGameProjects } from '@/queries/useGameProjects'
import { getGameProjectProgress, type GameProjectCreateInput } from '@/api/gameProjects'
import { tokens as t } from '@/styles/tokens'

const useStyles = createStyles(({ css }) => ({
  page: css`max-width: 1280px; margin: 0 auto;`,
  header: css`display:flex; justify-content:space-between; align-items:flex-start; gap:20px; margin-bottom:24px; @media(max-width:680px){flex-direction:column;}`,
  title: css`margin:0; color:${t.text}; font-size:${t.text2xl}; letter-spacing:-.03em;`,
  subtitle: css`max-width:64ch; margin:7px 0 0; color:${t.textTertiary}; line-height:1.65;`,
  guide: css`display:grid; grid-template-columns:repeat(3,1fr); gap:1px; overflow:hidden; margin-bottom:22px; border:1px solid var(--border); border-radius:${t.radiusLg}px; background:var(--border); @media(max-width:760px){grid-template-columns:1fr;}`,
  guideItem: css`padding:16px 18px; background:var(--card); strong{display:block;color:${t.text};} span{display:block;margin-top:4px;color:${t.textMuted};font-size:${t.textSm};line-height:1.5;}`,
  grid: css`display:grid; grid-template-columns:repeat(auto-fill,minmax(300px,1fr)); gap:16px;`,
  card: css`display:flex; flex-direction:column; min-height:250px; padding:20px; text-align:left; border:1px solid var(--border); border-radius:${t.radiusLg}px; background:var(--card); cursor:pointer; transition:.18s ease; &:hover{transform:translateY(-2px);border-color:color-mix(in srgb,var(--primary) 45%,var(--border));box-shadow:${t.elevation1};} &:focus-visible{outline:3px solid color-mix(in srgb,var(--primary) 28%,transparent);outline-offset:2px;}`,
  cardTop: css`display:flex; justify-content:space-between; gap:12px;`,
  icon: css`display:grid;place-items:center;width:42px;height:42px;border-radius:12px;color:var(--primary);background:var(--primary-subtle);`,
  cardTitle: css`margin:16px 0 6px;color:${t.text};font-size:${t.textLg};`,
  cardText: css`margin:0;color:${t.textTertiary};font-size:${t.textSm};line-height:1.6;`,
  progress: css`margin-top:auto;padding-top:18px;`,
  cardFoot: css`display:flex;justify-content:space-between;align-items:center;margin-top:8px;color:${t.textMuted};font-size:12px;`,
  feedback: css`margin:28px 0;`,
}))

export default function GameProjectListPage() {
  const { styles } = useStyles()
  const navigate = useNavigate()
  const projectsQuery = useGameProjects()
  const createMutation = useCreateGameProject()
  const primaryStyles = usePrimaryButtonStyle()
  const [open, setOpen] = useState(false)
  const [form] = Form.useForm<GameProjectCreateInput>()

  const create = async () => {
    const values = await form.validateFields()
    const project = await createMutation.mutateAsync(values)
    setOpen(false)
    form.resetFields()
    await navigate(`/games/${project.project.id}`)
  }

  return <main className={styles.page}>
    <header className={styles.header}>
      <div><h1 className={styles.title}>游戏项目</h1><p className={styles.subtitle}>从一个游戏想法开始，在同一个项目里配置规则、人物、故事和视听内容。试玩确认后，系统才会生成可发布版本。</p></div>
      <PrimaryButton size="large" icon={<PlusIcon size={17}/>} onClick={() => { setOpen(true) }}>新建游戏</PrimaryButton>
    </header>
    <section className={styles.guide} aria-label="游戏制作流程">
      <div className={styles.guideItem}><strong>1. 建立游戏</strong><span>先写清题材、玩家要做什么，以及怎样算成功。</span></div>
      <div className={styles.guideItem}><strong>2. 填入内容</strong><span>逐步关联规则、Agent 人物、事件、场景和声音。</span></div>
      <div className={styles.guideItem}><strong>3. 试玩发布</strong><span>先试玩检查；发布时系统自动生成固定版本。</span></div>
    </section>
    {projectsQuery.isLoading ? <div className={styles.feedback}><Skeleton active paragraph={{ rows: 5 }}/></div> : projectsQuery.isError ? <Alert className={styles.feedback} showIcon type="error" title="暂时无法读取游戏项目" description="请检查网络后重试。已保存的项目不会丢失。" action={<PrimaryButton onClick={() => { void projectsQuery.refetch() }}>重试</PrimaryButton>}/> : projectsQuery.data?.length ? <section className={styles.grid} aria-label="已有游戏">
      {projectsQuery.data.map((detail) => {
        const project = detail.project
        const progress = getGameProjectProgress(detail)
        return <button className={styles.card} type="button" key={project.id} onClick={() => { void navigate(`/games/${project.id}`) }}>
          <div className={styles.cardTop}><span className={styles.icon}><GameControllerIcon size={22}/></span><Tag>{detail.releases.length > 0 ? '已发布' : '制作中'}</Tag></div>
          <h2 className={styles.cardTitle}>{project.name}</h2><p className={styles.cardText}>{project.description || '还没有填写游戏简介。'}</p>
          <div className={styles.progress}><Progress percent={progress.percent} showInfo={false}/></div>
          <div className={styles.cardFoot}><span>已完成 {progress.count}/6 步</span><span>继续制作 <ArrowRightIcon size={14}/></span></div>
        </button>
      })}
    </section> : <Empty description="还没有游戏项目，先创建第一个游戏吧"/>}
    <Modal title="新建游戏" open={open} onCancel={() => { setOpen(false) }} onOk={() => { void create() }} okText="创建并开始制作" cancelText="取消" confirmLoading={createMutation.isPending} okButtonProps={{className: primaryStyles.root}}>
      <Form form={form} layout="vertical" requiredMark={false}>
        <Form.Item name="name" label="游戏名称" rules={[{required:true,message:'请填写游戏名称'}]}><Input placeholder="例如：Speeding"/></Form.Item>
        <Form.Item name="description" label="一句话简介" rules={[{required:true,message:'请说明这是一个怎样的游戏'}]}><Input.TextArea rows={3} placeholder="玩家将面对什么局面？"/></Form.Item>
        <Form.Item name="genre" label="题材" rules={[{required:true,message:'请填写题材'}]}><Input placeholder="例如：财富与权力策略叙事"/></Form.Item>
        <Form.Item name="playGoal" label="玩家目标" rules={[{required:true,message:'请说明玩家要达成什么目标'}]}><Input.TextArea rows={3} placeholder="玩家最终需要完成什么？"/></Form.Item>
      </Form>
    </Modal>
  </main>
}
