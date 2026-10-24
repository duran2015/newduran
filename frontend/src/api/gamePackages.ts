import apiClient from './client'

export interface GameRegistrySummary {
  metricCount: number
  actionCount: number
  eventCount: number
  endingCount?: number
}

export interface GamePackageSummary {
  id: string
  key: string
  version: string
  displayName: string
  description?: string
  status: string
  enabled?: boolean
  contentHash: string
  registry: GameRegistrySummary
}

export interface GameMetricDefinition {
  id: string
  name?: string
  displayName?: string
  description?: string
  initialValue?: unknown
}

export interface GameActionDefinition {
  id: string
  name?: string
  displayName?: string
  description?: string
  exampleInput?: Record<string, unknown>
}

export interface GameEventDefinition {
  id: string
  name?: string
  displayName?: string
  description?: string
}

export interface GameEndingDefinition {
  id: string
  name?: string
  displayName?: string
  description?: string
}

export interface GamePackageManifest {
  schemaVersion: string
  package: {
    key: string
    version: string
    displayName: string
    description?: string
    author?: string
    license?: string
  }
  registry: {
    metrics: GameMetricDefinition[]
    actions: GameActionDefinition[]
    events: GameEventDefinition[]
    endings: GameEndingDefinition[]
  }
  acceptanceScenarios?: {
    id: string
    name?: string
    description?: string
    initialState?: Record<string, unknown>
    commands: GameCommand[]
  }[]
}

export interface GamePackageDetail {
  package: GamePackageSummary
  manifest: GamePackageManifest
}

export interface GamePackageList {
  items: GamePackageSummary[]
  total: number
}

export interface GameCommand {
  actionId: string
  input?: Record<string, unknown>
}

export interface GameStateDiff {
  metricId?: string
  path?: string
  key?: string
  before?: unknown
  after?: unknown
  oldValue?: unknown
  newValue?: unknown
}

export interface GameTestRunStep {
  sequence: number
  command: GameCommand
  event?: { id?: string; sequence?: number; actionId?: string; payload?: unknown } | string
  stateDiff?: GameStateDiff[] | Record<string, unknown>
  state: Record<string, unknown>
}

export interface GameTestRunResult {
  packageId: string
  packageVersion: string
  initialState: Record<string, unknown>
  finalState: Record<string, unknown>
  steps: GameTestRunStep[]
  passed: boolean
}

export const gamePackagesApi = {
  list: () => apiClient.get('/api/v1/admin/game-packages'),
  installAcceptanceFixtures: () => apiClient.post('/api/v1/admin/game-packages/acceptance-data'),
  detail: (id: string) => apiClient.get(`/api/v1/admin/game-packages/${id}`),
  setEnabled: (id: string, enabled: boolean) =>
    apiClient.patch(`/api/v1/admin/game-packages/${id}/enabled`, { enabled }),
  testRun: (id: string, commands: GameCommand[], initialState?: Record<string, unknown>) =>
    apiClient.post(`/api/v1/admin/game-packages/${id}/test-runs`, { commands, initialState }),
}
