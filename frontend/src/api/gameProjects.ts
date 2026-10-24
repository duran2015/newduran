import apiClient from './client'
import type { GameEndingDefinition } from './gamePackages'

export interface GameProject {
  id: string
  name: string
  key: string
  description: string
  genre: string
  playGoal: string
  status: 'draft' | 'published'
  revision: number
  createdAt: string
  updatedAt: string
}

export interface GameProjectCreateInput {
  name: string
  description: string
  genre: string
  playGoal: string
}

export type GameStateComparison = 'gt' | 'gte' | 'lt' | 'lte' | 'eq' | 'ne'
export type GameStateChangeMethod = 'increment' | 'decrement' | 'set'

export interface GameProjectMetric {
  id: string
  name: string
  description?: string
  type: 'integer' | 'number'
  initial: number
  minimum?: number
  maximum?: number
  displayUnit?: string
}

export interface GameActionCondition {
  target: string
  operator: GameStateComparison
  value: number
}

export interface GameActionEffect {
  target: string
  operation: GameStateChangeMethod
  value: number
}

export interface GameProjectAction {
  id: string
  name: string
  description?: string
  conditionMode: 'all' | 'any'
  conditions: GameActionCondition[]
  effects: GameActionEffect[]
  timeCost: number
  emits?: string
}

export interface GameProjectObjective {
  id: string
  name: string
  description?: string
  kind: 'main' | 'optional'
  issuerCharacterId?: string
  startsActive: boolean
  dependsOn: string[]
  conditionMode: 'all' | 'any'
  conditions: GameActionCondition[]
  /** Optional P0 task-lifecycle fields. Older servers safely ignore omitted values. */
  failureConditionMode?: 'all' | 'any'
  failureConditions?: GameActionCondition[]
  deadlineTurns?: number
  completedEffects?: GameActionEffect[]
  failedEffects?: GameActionEffect[]
  promiseText?: string
}

export type GameEventCategory = 'system' | 'route' | 'personal' | 'environment'

export interface GameProjectEvent {
  sceneId?: string
  endReason?: string
  id: string
  name: string
  description?: string
  category: GameEventCategory
  displayContent: string
  conditionMode: 'all' | 'any'
  conditions: GameActionCondition[]
  participantIds: string[]
  /** Canonical backend field; participantIds remains supported for older drafts. */
  participantCharacterIds?: string[]
  effects: GameActionEffect[]
  advancesTime: boolean
  timeAdvance: number
  priority: number
  once: boolean
  cooldownTurns: number
  exclusiveGroup?: string
  nextEventIds: string[]
  endsGame: boolean
}

export interface PreviewGameEventsInput {
  state: Record<string, number>
  currentTurn: number
  triggeredAtTurn?: Record<string, number>
  candidateEventIds?: string[]
}

export interface GameEventPreviewItem {
  eventId: string
  name: string
  category: string
  eligible: boolean
  selected: boolean
  reasons: string[]
  checks: { target: string; operator: string; expected: unknown; actual: unknown; matched: boolean; reason: string }[]
  before: Record<string, number>
  after: Record<string, number>
  stateDiff: { metricId: string; before: unknown; after: unknown }[]
  timeAdvance: number
  nextEventIds?: string[]
  endsGame: boolean
  priority: number
  exclusiveGroup?: string
}

export interface PreviewGameEventsResult {
  currentTurn: number
  items: GameEventPreviewItem[]
  selectedIds: string[]
  summary: string
}

export type GameSessionStatus = 'running' | 'paused' | 'completed'

export interface GameSessionParticipant {
  characterId: string
  displayName: string
  role?: string
  status?: string
}

export interface GameSessionStateChange {
  metricId: string
  before: unknown
  after: unknown
}

export interface GameSessionEventSummary {
  eventId?: string
  name: string
  displayContent?: string
  participants?: (GameSessionParticipant | string)[]
  stateDiff?: GameSessionStateChange[]
  timeAdvance?: number
  endsGame?: boolean
}

export interface GameSessionTimelineItem {
  id: string
  kind: string
  name?: string
  status?: string
  actorId?: string
  input?: Record<string, unknown>
  output?: Record<string, unknown>
  occurredAt: string
}

/** One playable draft run. The editor uses plain-language labels for this data. */
export interface GameSession {
  id: string
  runId?: string
  projectId: string
  projectRevision: number
  status: GameSessionStatus
  directorCharacterId?: string
  currentTurn: number
  state: Record<string, number>
  latestEvent?: GameSessionEventSummary | null
  participants?: GameSessionParticipant[]
  timeline: GameSessionTimelineItem[]
  createdAt: string
  updatedAt: string
}

export interface GameSessionDetail { session: GameSession }

export type GameCharacterType = 'director' | 'npc' | 'observer'

/** A game-owned role that references an Agent Hub agent by name; it never copies the agent. */
export interface GameProjectCharacter {
  id: string
  agentId: number
  displayName: string
  role: string
  responsibilities?: string
  characterType: 'director' | 'npc' | 'observer'
  visibleMetricIds: string[]
  executableActionIds: string[]
  advisoryActionIds: string[]
  knownInformation: string[]
  informationScope?: string
  knowledgeRefs: string[]
  relationIds: number[]
  permissions: string[]
}

export type GameProjectAssetKind = 'background' | 'portrait' | 'music' | 'sound'

export interface GameProjectAsset {
  id: string
  projectId: string
  name: string
  kind: GameProjectAssetKind
  originalName: string
  mimeType: string
  sizeBytes: number
  width?: number
  height?: number
  durationMs?: number
  contentHash: string
  url: string
  usedBy?: string[]
  available?: boolean
  problem?: string
  createdAt: string
}

export interface GameProjectAssetList { items: GameProjectAsset[]; total: number }

export interface GameProjectSceneCharacter {
  id: string
  characterId: string
  portraitAssetId: string
  emotionPortraitAssetIds?: Partial<Record<'calm' | 'wary' | 'tense' | 'angry' | 'elated' | 'grieving', string>>
  x: number
  y: number
  scale: number
  layer: number
  visible: boolean
}

export interface GameProjectScene {
  id: string
  name: string
  description?: string
  order: number
  backgroundAssetId?: string
  musicAssetId?: string
  soundEffectAssetIds: string[]
  characters: GameProjectSceneCharacter[]
}

export interface GameCharacterContextPreview {
  characterId: string
  displayName: string
  identity: string
  responsibilities?: string
  agentName: string
  agentAvailable: boolean
  availableSummary: string
  visibleWorldState: string[]
  canExecute: string[]
  canSuggest: string[]
  knownInformation: string[]
  knowledgeSummary: string
  relationships: string[]
  limits: string[]
}

export interface PreviewGameActionInput { actionId: string; state: Record<string, number>; input?: Record<string, unknown>; requestKey?: string }
export interface PreviewGameActionResult {
  requestKey?: string
  actionId: string
  allowed: boolean
  reasons: string[]
  checks: { target: string; operator: string; expected: unknown; actual: unknown; matched: boolean; reason: string }[]
  before: Record<string, number>
  after: Record<string, number>
  stateDiff: { metricId: string; before: unknown; after: unknown }[]
}

export interface GameProjectDraft { metrics: GameProjectMetric[]; actions: GameProjectAction[]; endings: GameEndingDefinition[]; objectives?: GameProjectObjective[]; npcs: GameProjectCharacter[]; events: GameProjectEvent[]; scenes: GameProjectScene[] }
export interface GameProjectRelease { id:string; projectId:string; version:string; projectRevision:number; packageId:string; contentHash:string; releaseNotes?:string; publishedBy?:string; publishedAt:string }
export interface GamePublishCheckItem { key:string; label:string; ready:boolean; blocking:boolean; detail:string; step?:string }
export interface GamePublishCheck { canPublish:boolean; items:GamePublishCheckItem[] }
export interface GameProjectDetail { project: GameProject; draft: GameProjectDraft; releases: GameProjectRelease[] }
export interface GameProjectReleaseDetail { release: GameProjectRelease; projectName: string; summary: { worldState: number; actions: number; characters: number; events: number; scenes: number } }
export interface GameProjectList { items: GameProject[]; total: number }
export interface GameProjectUpdateInput extends GameProjectCreateInput {
  revision: number
  draft?: GameProjectDraft
}

export function getGameProjectProgress(detail: Pick<GameProjectDetail, 'project' | 'draft' | 'releases'>) {
  const { project, draft, releases } = detail
  const completed = [
    Boolean(project.name.trim() && project.description.trim() && project.genre.trim() && project.playGoal.trim()),
    draft.metrics.length > 0 && draft.actions.length > 0,
    draft.npcs.length > 0,
    draft.events.length > 0,
    draft.scenes.length > 0,
    releases.length > 0,
  ]
  return { completed, count: completed.filter(Boolean).length, percent: Math.round((completed.filter(Boolean).length / completed.length) * 100) }
}

function safeProjectKey(name: string) {
  const latin = name.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  return `${latin || 'game'}-${Date.now().toString(36)}`
}

export const gameProjectsApi = {
  list: () => apiClient.get('/api/v1/admin/game-projects'),
  detail: (id: string) => apiClient.get(`/api/v1/admin/game-projects/${id}`),
  publishCheck: (id: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/publish-check`),
  releaseDetail: (id: string, releaseId: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/releases/${releaseId}`),
  restoreRelease: (id: string, releaseId: string, revision: number) => apiClient.post(`/api/v1/admin/game-projects/${id}/releases/${releaseId}/restore`, { mode: 'replace-draft', revision }),
  publish: (id: string, input: {version:string;releaseNotes:string}) => apiClient.post(`/api/v1/admin/game-projects/${id}/publish`, input),
  create: (input: GameProjectCreateInput) => apiClient.post('/api/v1/admin/game-projects', { key: safeProjectKey(input.name), ...input }),
  update: (id: string, input: GameProjectUpdateInput) => apiClient.put(`/api/v1/admin/game-projects/${id}`, input),
  previewAction: (id: string, input: PreviewGameActionInput) => apiClient.post(`/api/v1/admin/game-projects/${id}/preview-action`, input),
  previewEvents: (id: string, input: PreviewGameEventsInput) => apiClient.post(`/api/v1/admin/game-projects/${id}/preview-events`, input),
  previewCharacterContext: (id: string, characterId: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/characters/${characterId}/context-preview`),
  listSessions: (id: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/sessions`),
  createSession: (id: string) => apiClient.post(`/api/v1/admin/game-projects/${id}/sessions`),
  getSession: (id: string, sessionId: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/sessions/${sessionId}`),
  advanceSession: (id: string, sessionId: string, input: { requestKey: string; actorCharacterId?: string }) => apiClient.post(`/api/v1/admin/game-projects/${id}/sessions/${sessionId}/advance`, input),
  pauseSession: (id: string, sessionId: string, input: { actorCharacterId?: string }) => apiClient.post(`/api/v1/admin/game-projects/${id}/sessions/${sessionId}/pause`, input),
  resumeSession: (id: string, sessionId: string, input: { actorCharacterId?: string }) => apiClient.post(`/api/v1/admin/game-projects/${id}/sessions/${sessionId}/resume`, input),
  endSession: (id: string, sessionId: string, input: { actorCharacterId?: string }) => apiClient.post(`/api/v1/admin/game-projects/${id}/sessions/${sessionId}/end`, input),
  listAssets: (id: string) => apiClient.get(`/api/v1/admin/game-projects/${id}/assets`),
  uploadAsset: (id: string, input: { file: File; name: string; kind: GameProjectAssetKind; onProgress?: (percent: number) => void }) => {
    const body = new FormData()
    body.append('file', input.file)
    body.append('name', input.name)
    body.append('kind', input.kind)
    return apiClient.post(`/api/v1/admin/game-projects/${id}/assets`, body, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
      onUploadProgress: (event) => { input.onProgress?.(event.total ? Math.round((event.loaded / event.total) * 100) : 0) },
    })
  },
  replaceAsset: (id: string, assetId: string, input: { file: File; name: string; onProgress?: (percent: number) => void }) => {
    const body = new FormData()
    body.append('file', input.file)
    body.append('name', input.name)
    return apiClient.put(`/api/v1/admin/game-projects/${id}/assets/${assetId}`, body, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
      onUploadProgress: (event) => { input.onProgress?.(event.total ? Math.round((event.loaded / event.total) * 100) : 0) },
    })
  },
  deleteAsset: (id: string, assetId: string) => apiClient.delete(`/api/v1/admin/game-projects/${id}/assets/${assetId}`),
  createScene: (id: string, input: { revision: number; scene: GameProjectScene }) => apiClient.post(`/api/v1/admin/game-projects/${id}/scenes`, input),
  updateScene: (id: string, sceneId: string, input: { revision: number; scene: GameProjectScene }) => apiClient.put(`/api/v1/admin/game-projects/${id}/scenes/${sceneId}`, input),
  deleteScene: (id: string, sceneId: string, revision: number) => apiClient.delete(`/api/v1/admin/game-projects/${id}/scenes/${sceneId}`, { params: { revision } }),
}
