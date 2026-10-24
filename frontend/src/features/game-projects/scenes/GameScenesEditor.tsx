import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, App, Button, Empty, Input, Modal, Progress, Select, Segmented, Switch, Tag, Upload } from 'antd'
import type { UploadFile } from 'antd'
import {
  ArrowClockwiseIcon, ArrowDownIcon, ArrowUpIcon, CopyIcon, EyeIcon, EyeSlashIcon, ImageIcon,
  MusicNoteIcon, PauseIcon, PlayIcon, PlusIcon, SpeakerHighIcon, TrashIcon, UploadSimpleIcon,
} from '@phosphor-icons/react'
import { createStyles } from 'antd-style'
import PrimaryButton from '@/components/PrimaryButton'
import { tokens as t } from '@/styles/tokens'
import type { GameProjectAsset, GameProjectAssetKind, GameProjectCharacter, GameProjectScene, GameProjectSceneCharacter } from '@/api/gameProjects'
import { authFetchBlob } from '@/api/agent-chat'

export type GameMediaType = GameProjectAssetKind
export type GamePortraitPosition = 'left' | 'center' | 'right'

interface Props {
  assets: GameProjectAsset[]
  scenes: GameProjectScene[]
  characters: GameProjectCharacter[]
  onChange: (scenes: GameProjectScene[]) => void
  onSave: (scenes: GameProjectScene[]) => Promise<void> | void
  onUpload: (file: File, type: GameMediaType, onProgress: (percent: number) => void) => Promise<void>
  onReplaceAsset: (asset: GameProjectAsset, file: File, onProgress: (percent: number) => void) => Promise<void>
  onDeleteAsset: (asset: GameProjectAsset) => Promise<void> | void
  saving?: boolean
}

const mediaLabels: Record<GameMediaType, string> = { background: '背景图', portrait: '人物立绘', music: '背景音乐', sound: '音效' }
const positionLabels: Record<GamePortraitPosition, string> = { left: '左侧', center: '中间', right: '右侧' }
const emotionLabels = { calm: '平静', wary: '警惕', tense: '紧张', angry: '愤怒', elated: '振奋', grieving: '悲伤' } as const
const acceptByType: Record<GameMediaType, string> = {
  background: 'image/png,image/jpeg,image/webp', portrait: 'image/png,image/webp',
  music: 'audio/mpeg,audio/wav,audio/ogg', sound: 'audio/mpeg,audio/wav,audio/ogg',
}
const maxSizeByType: Record<GameMediaType, number> = { background: 10, portrait: 10, music: 30, sound: 30 }

function requestErrorMessage(error: unknown, fallback: string) {
  if (typeof error !== 'object' || error === null || !('response' in error)) return fallback
  const response = (error as { response?: { data?: { error?: string; message?: string } } }).response
  return response?.data?.error ?? response?.data?.message ?? fallback
}

const useStyles = createStyles(({ css }) => ({
  intro: css`margin:20px 0 16px;`,
  workspace: css`display:grid;grid-template-columns:230px minmax(0,1fr) 270px;min-height:650px;border:1px solid var(--border);border-radius:${t.radius}px;overflow:hidden;background:var(--background);@media(max-width:1050px){grid-template-columns:210px minmax(0,1fr);}.ant-input,.ant-select-selector{font-size:14px;}@media(max-width:720px){display:block;min-height:auto;border:0;overflow:visible;}`,
  mobileMode: css`display:none;margin-bottom:12px;@media(max-width:720px){display:flex;}`,
  pane: css`min-width:0;padding:14px;border-right:1px solid var(--border);@media(max-width:720px){display:none!important;padding:0;border:0;&[data-mobile-active="true"]{display:block!important;}}`,
  settingsPane: css`min-width:0;padding:14px;@media(max-width:1050px){grid-column:1/-1;border-top:1px solid var(--border);}@media(max-width:720px){display:none!important;padding:0;border:0;&[data-mobile-active="true"]{display:block!important;}}`,
  paneTitle: css`display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:12px;strong{font-size:${t.textBase};}`,
  sceneList: css`display:grid;gap:6px;`,
  sceneItem: css`display:flex;align-items:center;gap:8px;width:100%;padding:9px 10px;border:0;border-radius:9px;color:${t.textSecondary};text-align:left;background:transparent;cursor:pointer;&[data-selected="true"]{color:${t.text};background:var(--secondary);}&:focus-visible{outline:2px solid var(--primary);outline-offset:1px;}`,
  sceneName: css`min-width:0;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;`,
  sceneActions: css`display:flex;gap:2px;margin-top:12px;padding-top:12px;border-top:1px solid var(--border);button{flex:1;}`,
  canvasPane: css`min-width:0;padding:16px;@media(max-width:720px){display:none!important;padding:0;&[data-mobile-active="true"]{display:block!important;}}`,
  canvasHead: css`display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:12px;@media(max-width:520px){align-items:stretch;flex-direction:column;}`,
  nameBlock: css`min-width:0;flex:1;.ant-input{font-weight:650;font-size:${t.textLg};}.ant-input-textarea{margin-top:6px;}`,
  canvas: css`position:relative;aspect-ratio:16/9;min-height:300px;overflow:hidden;border-radius:${t.radius}px;background:color-mix(in srgb,var(--secondary) 65%,var(--background));outline:1px solid color-mix(in srgb,var(--foreground) 10%,transparent);outline-offset:-1px;@media(max-width:720px){min-height:230px;}`,
  background: css`position:absolute;inset:0;width:100%;height:100%;object-fit:cover;`,
  noBackground: css`position:absolute;inset:0;display:grid;place-items:center;color:${t.textMuted};text-align:center;span{display:grid;gap:7px;place-items:center;}`,
  portrait: css`position:absolute;bottom:0;display:grid;place-items:end center;width:36%;height:92%;transform-origin:bottom center;pointer-events:none;&[data-position="left"]{left:1%;}&[data-position="center"]{left:32%;}&[data-position="right"]{right:1%;}img{max-width:100%;max-height:100%;object-fit:contain;filter:drop-shadow(0 5px 10px rgb(0 0 0 / 18%));}`,
  previewHint: css`display:flex;align-items:center;justify-content:space-between;gap:10px;margin-top:9px;color:${t.textMuted};font-size:${t.textSm};`,
  settingsSection: css`padding:0 0 14px;margin:0 0 14px;border-bottom:1px solid var(--border);&:last-child{border:0;margin:0;}h4{margin:0 0 9px;color:${t.text};font-size:${t.textSm};}`,
  portraitRows: css`display:grid;gap:8px;`,
  portraitRow: css`padding:9px;border-radius:9px;background:var(--secondary);`,
  portraitTop: css`display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:8px;strong{font-size:${t.textSm};}`, 
  portraitControls: css`display:grid;grid-template-columns:1fr 82px;gap:7px;`,
  scaleRow: css`display:flex;align-items:center;gap:7px;margin-top:7px;span{color:${t.textMuted};font-size:12px;white-space:nowrap;}input{width:100%;accent-color:var(--primary);}`,
  soundRow: css`display:flex;align-items:center;gap:7px;margin-top:7px;.ant-select{min-width:0;flex:1;}`,
  saveRow: css`display:flex;justify-content:flex-end;margin-top:16px;`,
  assetToolbar: css`display:flex;align-items:center;gap:9px;margin:16px 0 12px;@media(max-width:620px){align-items:stretch;flex-direction:column;}`,
  assetCount: css`margin-left:auto;color:${t.textMuted};font-size:${t.textSm};@media(max-width:620px){margin-left:0;}`,
  assetGrid: css`display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:10px;`,
  asset: css`min-width:0;overflow:hidden;border:1px solid var(--border);border-radius:${t.radius}px;background:var(--background);`,
  assetPreview: css`position:relative;display:grid;place-items:center;aspect-ratio:16/9;overflow:hidden;color:${t.textMuted};background:var(--secondary);img{width:100%;height:100%;object-fit:cover;}`,
  assetBody: css`display:flex;align-items:center;gap:6px;padding:9px 9px 4px;`,
  assetMeta: css`display:flex;align-items:center;gap:5px;min-height:30px;padding:0 9px 9px;color:${t.textMuted};font-size:12px;.ant-tag{margin:0;max-width:100%;overflow:hidden;text-overflow:ellipsis;}`,
  assetName: css`min-width:0;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:${t.textSm};`,
  uploadProgress: css`margin-top:10px;padding:10px;border-radius:9px;background:var(--secondary);`,
  empty: css`padding:26px 12px;border:1px dashed var(--border);border-radius:${t.radius}px;`,
}))

function newSceneId() {
  return `scene-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`
}

function useAuthorizedAssetUrls(assets: GameProjectAsset[]) {
  const direct = useMemo(() => Object.fromEntries(assets.filter((asset) => !asset.url.startsWith('/api/')).map((asset) => [asset.id, asset.url])), [assets])
  const [loaded, setLoaded] = useState<Record<string, string>>({})
  const [failed, setFailed] = useState<Set<string>>(new Set())
  useEffect(() => {
    let active = true
    const objectUrls: string[] = []
    void Promise.all(assets.filter((asset) => asset.url.startsWith('/api/')).map(async (asset) => {
      try {
        const blob = await authFetchBlob(asset.url)
        const url = URL.createObjectURL(blob); objectUrls.push(url)
        if (active) setLoaded((current) => ({ ...current, [asset.id]: url }))
      } catch {
        if (active) setFailed((current) => new Set(current).add(asset.id))
      }
    }))
    return () => { active = false; objectUrls.forEach((url) => { URL.revokeObjectURL(url) }) }
  }, [assets])
  return { resolved: { ...direct, ...loaded }, failed }
}

function portraitPosition(x: number): GamePortraitPosition {
  if (x < 35) return 'left'
  if (x > 65) return 'right'
  return 'center'
}

const portraitX: Record<GamePortraitPosition, number> = { left: 18, center: 50, right: 82 }

export default function GameScenesEditor({ assets, scenes, characters, onChange, onSave, onUpload, onReplaceAsset, onDeleteAsset, saving }: Props) {
  const { styles } = useStyles()
  const { message } = App.useApp()
  const [selectedSceneId, setSelectedSceneId] = useState(scenes[0]?.id)
  const [mediaType, setMediaType] = useState<GameMediaType>('background')
  const [mobileMode, setMobileMode] = useState<'scenes' | 'canvas' | 'settings'>('canvas')
  const [uploading, setUploading] = useState<{ name: string; percent: number }>()
  const [audioId, setAudioId] = useState<string>()
  const audioRef = useRef<HTMLAudioElement | undefined>(undefined)
  const selected: GameProjectScene | undefined = scenes.length === 0 ? undefined : (scenes.find((scene) => scene.id === selectedSceneId) ?? scenes[0])
  const assetMap = useMemo(() => new Map(assets.map((asset) => [asset.id, asset])), [assets])
  const assetUrls = useAuthorizedAssetUrls(assets)
  const filteredAssets = assets.filter((asset) => asset.kind === mediaType)
  const background = selected?.backgroundAssetId ? assetMap.get(selected.backgroundAssetId) : undefined

  const replaceScene = (patch: Partial<GameProjectScene>) => {
    if (!selected) return
    onChange(scenes.map((scene) => scene.id === selected.id ? { ...scene, ...patch } : scene))
  }
  const addScene = () => {
    const scene: GameProjectScene = { id: newSceneId(), name: `新场景 ${scenes.length + 1}`, order: scenes.length, characters: [], soundEffectAssetIds: [] }
    onChange([...scenes, scene]); setSelectedSceneId(scene.id); setMobileMode('canvas')
  }
  const copyScene = () => {
    if (!selected) return
    const copy = { ...selected, id: newSceneId(), name: `${selected.name}（副本）`, order: scenes.length, characters: selected.characters.map((item) => ({ ...item, emotionPortraitAssetIds: { ...item.emotionPortraitAssetIds }, id: `${item.characterId}-${newSceneId()}` })), soundEffectAssetIds: [...selected.soundEffectAssetIds] }
    onChange([...scenes, copy]); setSelectedSceneId(copy.id)
  }
  const deleteScene = () => {
    if (!selected) return
    Modal.confirm({ title: `删除“${selected.name}”？`, content: '这个场景的画面和声音设置会被删除，资源库中的原文件仍会保留。', okText: '删除场景', cancelText: '取消', okButtonProps: { danger: true }, onOk: () => { const next = scenes.filter((scene) => scene.id !== selected.id); onChange(next); setSelectedSceneId(next[0]?.id) } })
  }
  const moveScene = (direction: -1 | 1) => {
    if (!selected) return
    const from = scenes.findIndex((scene) => scene.id === selected.id); const to = from + direction
    if (to < 0 || to >= scenes.length) return
    const next = [...scenes]; [next[from], next[to]] = [next[to], next[from]]; onChange(next.map((scene, index) => ({ ...scene, order: index })))
  }
  const updatePortrait = (characterId: string, patch: Partial<GameProjectSceneCharacter>) => {
    if (!selected) return
    const existing = selected.characters.find((item) => item.characterId === characterId)
    const characters = existing
      ? selected.characters.map((item) => item.characterId === characterId ? { ...item, ...patch } : item)
      : [...selected.characters, { id: `${characterId}-${newSceneId()}`, characterId, portraitAssetId: '', x: 50, y: 100, scale: 1, layer: selected.characters.length + 1, visible: true, ...patch }]
    replaceScene({ characters })
  }
  const uploadProps = {
    accept: acceptByType[mediaType], showUploadList: false,
    beforeUpload: (file: UploadFile) => {
      const raw = file as unknown as File
      const maxSize = maxSizeByType[mediaType]
      if (raw.size > maxSize * 1024 * 1024) { message.error(`${mediaLabels[mediaType]}不能超过 ${maxSize}MB`); return Upload.LIST_IGNORE }
      setUploading({ name: raw.name, percent: 0 })
      void onUpload(raw, mediaType, (percent) => { setUploading({ name: raw.name, percent }) }).then(() => { message.success(`${mediaLabels[mediaType]}已加入资源库`) }).catch((error: unknown) => { message.error(requestErrorMessage(error, '上传失败，请检查文件后重试')) }).finally(() => { setUploading(undefined) })
      return Upload.LIST_IGNORE
    },
  }
  const replaceProps = (asset: GameProjectAsset) => ({
    accept: acceptByType[asset.kind], showUploadList: false,
    beforeUpload: (file: UploadFile) => {
      const raw = file as unknown as File
      const maxSize = maxSizeByType[asset.kind]
      if (raw.size > maxSize * 1024 * 1024) { message.error(`${mediaLabels[asset.kind]}不能超过 ${maxSize}MB`); return Upload.LIST_IGNORE }
      setUploading({ name: `正在替换 ${asset.name}`, percent: 0 })
      void onReplaceAsset(asset, raw, (percent) => { setUploading({ name: `正在替换 ${asset.name}`, percent }) })
        .then(() => { message.success(`“${asset.name}”已替换，使用它的场景会自动更新`) })
        .catch((error: unknown) => { message.error(requestErrorMessage(error, '替换失败，原文件没有改变')) })
        .finally(() => { setUploading(undefined) })
      return Upload.LIST_IGNORE
    },
  })
  const toggleAudio = (asset?: GameProjectAsset) => {
    if (!asset) return
    if (audioId === asset.id) { audioRef.current?.pause(); setAudioId(undefined); return }
    const url = assetUrls.resolved[asset.id]
    if (!url) { message.warning('这个文件暂时无法预览'); return }
    audioRef.current?.pause(); const audio = new Audio(url); audioRef.current = audio; setAudioId(asset.id)
    audio.addEventListener('ended', () => { setAudioId(undefined) }, { once: true }); void audio.play().catch(() => { setAudioId(undefined); message.warning('浏览器暂时无法播放这个文件') })
  }

  return <>
    <Alert className={styles.intro} showIcon type="info" title="在这里搭好玩家看到和听到的场景" description="图片和声音保存在这款游戏的资源库中；场景只选择使用哪些资源，后续替换时不需要改代码。"/>
    <Segmented className={styles.mobileMode} block value={mobileMode} options={[{ label: '场景', value: 'scenes' }, { label: '画面', value: 'canvas' }, { label: '设置', value: 'settings' }]} onChange={(value) => { setMobileMode(value as typeof mobileMode) }}/>
    <div className={styles.workspace}>
      <aside className={styles.pane} data-mobile-active={mobileMode === 'scenes'}>
        <div className={styles.paneTitle}><strong>场景顺序</strong><Button size="small" icon={<PlusIcon/>} onClick={addScene}>新建</Button></div>
        {scenes.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有场景"><Button onClick={addScene}>新建第一个场景</Button></Empty></div> : <div className={styles.sceneList}>{scenes.map((scene, index) => <button type="button" key={scene.id} className={styles.sceneItem} data-selected={selected?.id === scene.id} onClick={() => { setSelectedSceneId(scene.id); setMobileMode('canvas') }}><span>{index + 1}.</span><span className={styles.sceneName}>{scene.name}</span></button>)}</div>}
        {selected && <div className={styles.sceneActions}><Button size="small" aria-label="场景上移" icon={<ArrowUpIcon/>} disabled={scenes[0].id === selected.id} onClick={() => { moveScene(-1) }}/><Button size="small" aria-label="场景下移" icon={<ArrowDownIcon/>} disabled={scenes[scenes.length - 1].id === selected.id} onClick={() => { moveScene(1) }}/><Button size="small" aria-label="复制场景" icon={<CopyIcon/>} onClick={copyScene}/><Button size="small" danger aria-label="删除场景" icon={<TrashIcon/>} onClick={deleteScene}/></div>}
      </aside>

      <section className={styles.canvasPane} data-mobile-active={mobileMode === 'canvas'}>
        {selected ? <><div className={styles.canvasHead}><div className={styles.nameBlock}><Input aria-label="场景名称" value={selected.name} maxLength={60} onChange={(event) => { replaceScene({ name: event.target.value }) }}/><Input.TextArea aria-label="场景说明" autoSize={{ minRows: 1, maxRows: 2 }} value={selected.description} maxLength={240} placeholder="这个场景在故事中用于什么？" onChange={(event) => { replaceScene({ description: event.target.value }) }}/></div><Tag>预览</Tag></div>
          <div className={styles.canvas}>
            {background && assetUrls.resolved[background.id] ? <img className={styles.background} src={assetUrls.resolved[background.id]} alt=""/> : <div className={styles.noBackground}><span><ImageIcon size={34}/>{background ? '背景图暂时无法预览' : '从右侧选择背景图'}</span></div>}
            {[...selected.characters].sort((a, b) => a.layer - b.layer).filter((item) => item.visible && assetUrls.resolved[item.portraitAssetId]).map((item) => <div key={item.id} className={styles.portrait} data-position={portraitPosition(item.x)} style={{ zIndex: item.layer, transform: `scale(${item.scale})` }}><img src={assetUrls.resolved[item.portraitAssetId]} alt=""/></div>)}
          </div><div className={styles.previewHint}><span>画面比例 16:9</span><Button size="small" disabled={!selected.musicAssetId} icon={audioId && audioId === selected.musicAssetId ? <PauseIcon/> : <PlayIcon/>} onClick={() => { toggleAudio(selected.musicAssetId ? assetMap.get(selected.musicAssetId) : undefined) }}>{audioId && audioId === selected.musicAssetId ? '暂停音乐' : '预览音乐'}</Button></div></> : <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="新建一个场景后，可以开始搭画面"><PrimaryButton onClick={addScene}>新建场景</PrimaryButton></Empty></div>}
      </section>

      <aside className={styles.settingsPane} data-mobile-active={mobileMode === 'settings'}>
        <div className={styles.paneTitle}><strong>画面与声音</strong></div>
        {!selected ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先新建场景"/> : <>
          <section className={styles.settingsSection}><h4>背景图</h4><Select aria-label="选择背景图" allowClear value={selected.backgroundAssetId} placeholder="选择背景图" options={assets.filter((a) => a.kind === 'background').map((a) => ({ label: a.name, value: a.id }))} onChange={(value) => { replaceScene({ backgroundAssetId: value }) }}/></section>
          <section className={styles.settingsSection}><h4>人物立绘</h4>{characters.length === 0 ? <Alert type="warning" showIcon title="请先在第 3 步添加游戏人物"/> : <div className={styles.portraitRows}>{characters.map((character) => { const item = selected.characters.find((portrait) => portrait.characterId === character.id); const portraitOptions = assets.filter((a) => a.kind === 'portrait').map((a) => ({ label: a.name, value: a.id })); return <div key={character.id} className={styles.portraitRow}><div className={styles.portraitTop}><strong>{character.displayName}</strong><Switch size="small" aria-label={`显示${character.displayName}`} checked={item?.visible ?? false} checkedChildren={<EyeIcon/>} unCheckedChildren={<EyeSlashIcon/>} onChange={(visible) => { updatePortrait(character.id, { visible }) }}/></div><div className={styles.portraitControls}><Select aria-label={`${character.displayName}的立绘`} allowClear value={item?.portraitAssetId ?? undefined} placeholder="默认立绘" options={portraitOptions} onChange={(portraitAssetId) => { updatePortrait(character.id, { portraitAssetId, visible: Boolean(portraitAssetId) }) }}/><Select aria-label={`${character.displayName}的位置`} value={portraitPosition(item?.x ?? 50)} options={Object.entries(positionLabels).map(([value, label]) => ({ value, label }))} onChange={(position: GamePortraitPosition) => { updatePortrait(character.id, { x: portraitX[position] }) }}/></div>{item?.portraitAssetId && <><div className={styles.scaleRow}><span>大小</span><input aria-label={`${character.displayName}立绘大小`} type="range" min="60" max="140" value={Math.round(item.scale * 100)} onChange={(event) => { updatePortrait(character.id, { scale: Number(event.target.value) / 100 }) }}/><span>{Math.round(item.scale * 100)}%</span></div><details style={{ marginTop: 8 }}><summary style={{ cursor: 'pointer', fontSize: 12, color: 'var(--text-muted)' }}>按人物心情切换立绘</summary><div style={{ display: 'grid', gap: 6, marginTop: 8 }}>{Object.entries(emotionLabels).map(([mood, label]) => <Select key={mood} size="small" aria-label={`${character.displayName}${label}时的立绘`} allowClear value={item.emotionPortraitAssetIds?.[mood as keyof typeof emotionLabels]} placeholder={`${label}时继续使用默认立绘`} options={portraitOptions} onChange={(assetId) => { updatePortrait(character.id, { emotionPortraitAssetIds: { ...item.emotionPortraitAssetIds, [mood]: assetId } }) }}/>)}</div></details></>}</div> })}</div>}</section>
          <section className={styles.settingsSection}><h4>背景音乐</h4><div className={styles.soundRow}><Select aria-label="选择背景音乐" allowClear value={selected.musicAssetId} placeholder="可选" options={assets.filter((a) => a.kind === 'music').map((a) => ({ label: a.name, value: a.id }))} onChange={(value) => { replaceScene({ musicAssetId: value }) }}/><Button aria-label="播放背景音乐" icon={audioId === selected.musicAssetId ? <PauseIcon/> : <PlayIcon/>} disabled={!selected.musicAssetId} onClick={() => { toggleAudio(selected.musicAssetId ? assetMap.get(selected.musicAssetId) : undefined) }}/></div></section>
          <section className={styles.settingsSection}><h4>入场音效</h4><Select aria-label="选择入场音效" mode="multiple" value={selected.soundEffectAssetIds} placeholder="可选多个" options={assets.filter((a) => a.kind === 'sound').map((a) => ({ label: a.name, value: a.id }))} onChange={(soundEffectAssetIds) => { replaceScene({ soundEffectAssetIds }) }}/></section>
        </>}
      </aside>
    </div>

    <div className={styles.assetToolbar}><Segmented value={mediaType} options={Object.entries(mediaLabels).map(([value, label]) => ({ value, label }))} onChange={(value) => { setMediaType(value as GameMediaType) }}/><Upload {...uploadProps}><Button icon={<UploadSimpleIcon/>} loading={Boolean(uploading)}>上传{mediaLabels[mediaType]}</Button></Upload><span className={styles.assetCount}>{filteredAssets.length} 个可用资源</span></div>
    {uploading && <div className={styles.uploadProgress}><div>{uploading.name}</div><Progress percent={uploading.percent}/></div>}
    {filteredAssets.length === 0 ? <div className={styles.empty}><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={`还没有${mediaLabels[mediaType]}`}><Upload {...uploadProps}><Button icon={<UploadSimpleIcon/>}>上传第一个{mediaLabels[mediaType]}</Button></Upload></Empty></div> : <div className={styles.assetGrid}>{filteredAssets.map((asset) => { const usedBy = asset.usedBy ?? []; const unavailable = asset.available === false || assetUrls.failed.has(asset.id); return <article className={styles.asset} key={asset.id}><div className={styles.assetPreview}>{(asset.kind === 'background' || asset.kind === 'portrait') && assetUrls.resolved[asset.id] ? <img src={assetUrls.resolved[asset.id]} alt=""/> : unavailable ? <span>{asset.problem ?? '文件暂时无法预览'}</span> : asset.kind === 'music' ? <MusicNoteIcon size={26}/> : <SpeakerHighIcon size={26}/>}</div><div className={styles.assetBody}><span className={styles.assetName} title={asset.name}>{asset.name}</span>{asset.kind === 'music' || asset.kind === 'sound' ? <Button type="text" size="small" aria-label={`播放${asset.name}`} icon={audioId === asset.id ? <PauseIcon/> : <PlayIcon/>} onClick={() => { toggleAudio(asset) }}/> : null}<Upload {...replaceProps(asset)}><Button type="text" size="small" aria-label={`替换${asset.name}`} icon={<ArrowClockwiseIcon/>}/></Upload><Button type="text" size="small" danger aria-label={`删除${asset.name}`} icon={<TrashIcon/>} onClick={() => { Modal.confirm({ title: `删除“${asset.name}”？`, content: usedBy.length > 0 ? `“${usedBy.join('、')}”正在使用它，请先从场景中取消使用。` : '删除后无法恢复，已经发布的版本不会受到影响。', okText: '删除资源', cancelText: '取消', okButtonProps: { danger: true }, onOk: async () => { try { await onDeleteAsset(asset); message.success('资源已删除') } catch (error) { message.error(requestErrorMessage(error, '删除失败，请稍后重试')); throw error } } }) }}/></div><div className={styles.assetMeta}>{unavailable ? <Tag color="error">需要重新上传</Tag> : usedBy.length > 0 ? <Tag color="success" title={usedBy.join('、')}>{usedBy.length === 1 ? `${usedBy[0]}正在使用` : `${usedBy.length}个场景正在使用`}</Tag> : <Tag>尚未使用</Tag>}</div></article> })}</div>}
    <div className={styles.saveRow}><PrimaryButton size="large" loading={saving} disabled={scenes.length === 0} onClick={() => { void onSave(scenes) }}>保存场景与视听</PrimaryButton></div>
  </>
}
