import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { unwrapResponse } from '@/api/client'
import { playerGamesApi, type PlayerGameSession, type PlayerGameSource } from '@/api/playerGames'

const key = (projectId?: string, sessionId?: string) => ['player-game', projectId, sessionId]

export function usePublishedGame(releaseId?: string) {
  return useQuery({ queryKey: ['published-game', releaseId], queryFn: async () => unwrapResponse<{ game: { name: string; description: string }; releaseId: string; version: string }>(await playerGamesApi.release(releaseId ?? '')), enabled: Boolean(releaseId) })
}

export function usePlayerGameSession(projectId?: string, sessionId?: string, releaseId?: string) {
  return useQuery({
    queryKey: key(projectId, sessionId),
    queryFn: async () => unwrapResponse<PlayerGameSession>(await playerGamesApi.getSession(projectId ?? '', sessionId ?? '', releaseId)),
    enabled: Boolean(sessionId && (projectId || releaseId)),
  })
}

function useStoreSession() {
  const client = useQueryClient()
  return (session: PlayerGameSession) => client.setQueryData(key(session.projectId, session.id), session)
}

export function useCreatePlayerGameSession() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, source, startAt }: { projectId: string; source: PlayerGameSource; startAt?: { kind: 'beginning' | 'turn' | 'event'; turn?: number; eventId?: string } }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.createSession(projectId, source, startAt)),
    onSuccess: store,
  })
}

export function useResetPlayerGameSession() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId }: { projectId: string; sessionId: string; releaseId?: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.resetSession(projectId, sessionId, releaseId)),
    onSuccess: store,
  })
}

export function useInterpretPlayerAction() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId, text, requestKey, targetCharacterId }: { projectId: string; sessionId: string; releaseId?: string; text: string; requestKey: string; targetCharacterId?: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.interpret(projectId, sessionId, { text, requestKey, targetCharacterId }, releaseId)),
    onSuccess: store,
  })
}

export function useConfirmPlayerAction() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId, planId, requestKey }: { projectId: string; sessionId: string; releaseId?: string; planId: string; requestKey: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.confirm(projectId, sessionId, { planId, requestKey }, releaseId)),
    onSuccess: store,
  })
}

export function useRejectPlayerAction() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId, planId }: { projectId: string; sessionId: string; releaseId?: string; planId: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.reject(projectId, sessionId, { planId }, releaseId)),
    onSuccess: store,
  })
}

export function useAcceptPlayerObjective() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId, objectiveId, requestKey }: { projectId: string; sessionId: string; releaseId?: string; objectiveId: string; requestKey: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.acceptObjective(projectId, sessionId, { objectiveId, requestKey }, releaseId)),
    onSuccess: store,
  })
}

export function useRejectPlayerObjective() {
  const store = useStoreSession()
  return useMutation({
    mutationFn: async ({ projectId, sessionId, releaseId, objectiveId, requestKey }: { projectId: string; sessionId: string; releaseId?: string; objectiveId: string; requestKey: string }) =>
      unwrapResponse<PlayerGameSession>(await playerGamesApi.rejectObjective(projectId, sessionId, { objectiveId, requestKey }, releaseId)),
    onSuccess: store,
  })
}
