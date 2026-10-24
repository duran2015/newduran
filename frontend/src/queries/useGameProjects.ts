import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { unwrapResponse } from '@/api/client'
import { gameProjectsApi, type GameProjectAsset, type GameProjectAssetKind, type GameProjectAssetList, type GameProjectCreateInput, type GameProjectDetail, type GameProjectList, type GameProjectRelease, type GameProjectReleaseDetail, type GameProjectUpdateInput, type GamePublishCheck, type GameSession, type PreviewGameActionInput, type PreviewGameActionResult, type PreviewGameEventsInput, type PreviewGameEventsResult } from '@/api/gameProjects'

export type GameProjectListItem = GameProjectDetail

export function useGameProjects() {
  return useQuery<GameProjectListItem[]>({
    queryKey: ['game-projects'],
    queryFn: async () => {
      const projects = (unwrapResponse<GameProjectList>(await gameProjectsApi.list())).items
      return Promise.all(projects.map(async (project) => unwrapResponse<GameProjectDetail>(await gameProjectsApi.detail(project.id))))
    },
  })
}

export function useGameProject(id?: string) {
  return useQuery<GameProjectDetail>({
    queryKey: ['game-projects', id],
    queryFn: async () => unwrapResponse<GameProjectDetail>(await gameProjectsApi.detail(id ?? '')),
    enabled: Boolean(id),
  })
}

export function useCreateGameProject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: GameProjectCreateInput) => unwrapResponse<GameProjectDetail>(await gameProjectsApi.create(input)),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-projects'] }) },
  })
}

export function useUpdateGameProject(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: GameProjectUpdateInput) => unwrapResponse<GameProjectDetail>(await gameProjectsApi.update(id ?? '', input)),
    onSuccess: async (detail) => {
      queryClient.setQueryData(['game-projects', id], detail)
      await queryClient.invalidateQueries({ queryKey: ['game-projects'] })
    },
  })
}

export function useGameReleaseDetail(id?: string, releaseId?: string) {
  return useQuery({ queryKey: ['game-projects', id, 'releases', releaseId], queryFn: async () => unwrapResponse<GameProjectReleaseDetail>(await gameProjectsApi.releaseDetail(id ?? '', releaseId ?? '')), enabled: Boolean(id && releaseId) })
}

export function useRestoreGameRelease(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({ mutationFn: async ({ releaseId, revision }: { releaseId: string; revision: number }) => unwrapResponse<GameProjectDetail>(await gameProjectsApi.restoreRelease(id ?? '', releaseId, revision)), onSuccess: async (detail) => { queryClient.setQueryData(['game-projects', id], detail); await queryClient.invalidateQueries({ queryKey: ['game-projects', id] }) } })
}

export function useGamePublishCheck(id?: string) {
  return useQuery<GamePublishCheck>({ queryKey:['game-projects',id,'publish-check'], queryFn:async()=>unwrapResponse<GamePublishCheck>(await gameProjectsApi.publishCheck(id??'')), enabled:Boolean(id) })
}

export function usePublishGameProject(id?: string) {
  const queryClient=useQueryClient()
  return useMutation({ mutationFn:async(input:{version:string;releaseNotes:string})=>unwrapResponse<GameProjectRelease>(await gameProjectsApi.publish(id??'',input)), onSuccess:async()=>{await queryClient.invalidateQueries({queryKey:['game-projects',id]});await queryClient.invalidateQueries({queryKey:['game-projects',id,'publish-check']})} })
}

export function useGameProjectAssets(id?: string) {
  return useQuery<GameProjectAsset[]>({
    queryKey: ['game-projects', id, 'assets'],
    queryFn: async () => unwrapResponse<GameProjectAssetList>(await gameProjectsApi.listAssets(id ?? '')).items,
    enabled: Boolean(id),
  })
}

export function useUploadGameProjectAsset(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { file: File; name: string; kind: GameProjectAssetKind; onProgress?: (percent: number) => void }) =>
      unwrapResponse<GameProjectAsset>(await gameProjectsApi.uploadAsset(id ?? '', input)),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-projects', id, 'assets'] }) },
  })
}

export function useDeleteGameProjectAsset(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (assetId: string) => { await gameProjectsApi.deleteAsset(id ?? '', assetId) },
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-projects', id, 'assets'] }) },
  })
}

export function useReplaceGameProjectAsset(id?: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { assetId: string; file: File; name: string; onProgress?: (percent: number) => void }) =>
      unwrapResponse<GameProjectAsset>(await gameProjectsApi.replaceAsset(id ?? '', input.assetId, input)),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-projects', id, 'assets'] }) },
  })
}

export function usePreviewGameAction(id?: string) {
  return useMutation({
    mutationFn: async (input: PreviewGameActionInput) => unwrapResponse<PreviewGameActionResult>(await gameProjectsApi.previewAction(id ?? '', input)),
  })
}

export function usePreviewGameEvents(id?: string) {
  return useMutation({
    mutationFn: async (input: PreviewGameEventsInput) => unwrapResponse<PreviewGameEventsResult>(await gameProjectsApi.previewEvents(id ?? '', input)),
  })
}

function unwrapSession(response: Awaited<ReturnType<typeof gameProjectsApi.createSession>>) {
  return unwrapResponse<GameSession>(response)
}

export function useGameSessions(id?: string) {
  return useQuery<GameSession[]>({
    queryKey: ['game-projects', id, 'sessions'],
    queryFn: async () => unwrapResponse<GameSession[]>(await gameProjectsApi.listSessions(id ?? '')),
    enabled: Boolean(id),
  })
}

export function useGameSession(projectId?: string, sessionId?: string) {
  return useQuery<GameSession>({
    queryKey: ['game-projects', projectId, 'sessions', sessionId],
    queryFn: async () => unwrapResponse<GameSession>(await gameProjectsApi.getSession(projectId ?? '', sessionId ?? '')),
    enabled: Boolean(projectId && sessionId),
  })
}

function useRefreshGameSessions() {
  const queryClient = useQueryClient()
  return async (projectId: string, session?: GameSession) => {
    if (session) queryClient.setQueryData(['game-projects', projectId, 'sessions', session.id], session)
    await queryClient.invalidateQueries({ queryKey: ['game-projects', projectId, 'sessions'] })
  }
}

export function useCreateGameSession() {
  const refresh = useRefreshGameSessions()
  return useMutation({
    mutationFn: async (projectId: string) => unwrapSession(await gameProjectsApi.createSession(projectId)),
    onSuccess: async (session, projectId) => refresh(projectId, session),
  })
}

export function useAdvanceGameSession() {
  const refresh = useRefreshGameSessions()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, actorCharacterId, requestKey }: { projectId: string; sessionId: string; actorCharacterId?: string; requestKey: string }) =>
      unwrapResponse<GameSession>(await gameProjectsApi.advanceSession(projectId, sessionId, { actorCharacterId, requestKey })),
    onSuccess: async (session, variables) => refresh(variables.projectId, session),
  })
}

export function useTransitionGameSession(action: 'pause' | 'resume' | 'end') {
  const refresh = useRefreshGameSessions()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, actorCharacterId }: { projectId: string; sessionId: string; actorCharacterId?: string }) => {
      const response = action === 'pause'
        ? await gameProjectsApi.pauseSession(projectId, sessionId, { actorCharacterId })
        : action === 'resume'
          ? await gameProjectsApi.resumeSession(projectId, sessionId, { actorCharacterId })
          : await gameProjectsApi.endSession(projectId, sessionId, { actorCharacterId })
      return unwrapResponse<GameSession>(response)
    },
    onSuccess: async (session, variables) => refresh(variables.projectId, session),
  })
}
