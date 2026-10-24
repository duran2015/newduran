import apiClient from './client'

export type PlayerGameSource =
  | { source: 'draft'; projectRevision: number }
  | { source: 'release'; releaseId: string }

export interface PlayerStateItem {
  id: string
  name: string
  value: string
  change?: string
}

export interface PlayerSceneCharacter {
  characterId: string
  name: string
  portraitUrl?: string
  mood?: 'calm' | 'wary' | 'tense' | 'angry' | 'elated' | 'grieving' | string
  x: number
  y: number
  scale: number
  layer: number
  visible?: boolean
}

export interface PlayerScene {
  id: string
  name: string
  description?: string
  backgroundUrl?: string
  musicUrl?: string
  soundUrls?: string[]
  characters: PlayerSceneCharacter[]
}

export interface PlayerMessage {
  id: string
  speaker: string
  kind: 'player' | 'npc' | 'narrator' | 'system'
  content: string
  occurredAt: string
	characterId?: string
	targetCharacterId?: string
}

export interface PlayerActionPlan {
  id: string
  summary: string
  inputText?: string
  actionId?: string
  steps?: string[]
  affectedState?: string[]
  effects?: { metricId: string; before: unknown; after: unknown }[]
  warnings: string[]
  needsConfirmation: boolean
}

export interface PlayerGameObjective {
  id: string
  name: string
  description?: string
  kind: 'main' | 'optional'
  status: 'locked' | 'offered' | 'active' | 'completed' | 'failed' | 'rejected'
  issuer?: string
  deadlineTurn?: number
  promiseText?: string
}

export interface PlayerGameSession {
  id: string
  projectId: string
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed'
  source: 'draft' | 'release'
  projectRevision: number
  releaseId?: string
  game?: { name: string; description: string }
  currentTurn: number
  currentSceneId?: string
  stateLabels?: Record<string, string>
  state: Record<string, unknown>
  objectives?: PlayerGameObjective[]
  scenes: PlayerScene[]
  timeline: { label: string; content: string; speaker?: string; characterId?: string; targetCharacterId?: string; occurredAt: string; stateDiff?: { metricId: string; before: unknown; after: unknown }[] }[]
  pendingPlan?: PlayerActionPlan | null
  ending?: { title: string; summary: string } | null
}

export const playerGamesApi = {
  release: (releaseId: string) => apiClient.get(`/api/v1/games/${releaseId}`),
  createSession: (projectId: string, source: PlayerGameSource, startAt?: { kind: 'beginning' | 'turn' | 'event'; turn?: number; eventId?: string }) => {
    const startingPoint = {
      startTurn: startAt?.kind === 'turn' ? startAt.turn : 0,
      startEventId: startAt?.kind === 'event' ? startAt.eventId : undefined,
    }
    return source.source === 'release'
      ? apiClient.post(`/api/v1/games/${source.releaseId}/sessions`, startingPoint)
      : apiClient.post(`/api/v1/admin/game-projects/${projectId}/player-sessions`, { ...source, ...startingPoint })
  },
  resetSession: (projectId: string, sessionId: string, releaseId?: string) => apiClient.post(
    releaseId
      ? `/api/v1/games/${releaseId}/sessions/${sessionId}/reset`
      : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/reset`,
  ),
  getSession: (projectId: string, sessionId: string, releaseId?: string) => releaseId
    ? apiClient.get(`/api/v1/games/${releaseId}/sessions/${sessionId}`)
    : apiClient.get(`/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}`),
  interpret: (projectId: string, sessionId: string, input: { text: string; requestKey: string; targetCharacterId?: string }, releaseId?: string) =>
    apiClient.post(releaseId ? `/api/v1/games/${releaseId}/sessions/${sessionId}/interpret` : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/interpret`, input, { timeout: 120000 }),
  confirm: (projectId: string, sessionId: string, input: { planId: string; requestKey: string }, releaseId?: string) =>
    apiClient.post(releaseId ? `/api/v1/games/${releaseId}/sessions/${sessionId}/confirm` : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/confirm`, input, { timeout: 120000 }),
  reject: (projectId: string, sessionId: string, input: { planId: string }, releaseId?: string) =>
    apiClient.post(releaseId ? `/api/v1/games/${releaseId}/sessions/${sessionId}/reject` : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/reject`, input),
  acceptObjective: (projectId: string, sessionId: string, input: { objectiveId: string; requestKey: string }, releaseId?: string) =>
    apiClient.post(releaseId ? `/api/v1/games/${releaseId}/sessions/${sessionId}/objectives/accept` : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/objectives/accept`, input),
  rejectObjective: (projectId: string, sessionId: string, input: { objectiveId: string; requestKey: string }, releaseId?: string) =>
    apiClient.post(releaseId ? `/api/v1/games/${releaseId}/sessions/${sessionId}/objectives/reject` : `/api/v1/admin/game-projects/${projectId}/player-sessions/${sessionId}/objectives/reject`, input),
}
