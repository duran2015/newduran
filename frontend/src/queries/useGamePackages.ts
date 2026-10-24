import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { unwrapResponse } from '@/api/client'
import {
  gamePackagesApi,
  type GameCommand,
  type GamePackageDetail,
  type GamePackageList,
  type GameTestRunResult,
} from '@/api/gamePackages'

export function useGamePackages() {
  return useQuery<GamePackageList>({
    queryKey: ['game-packages'],
    queryFn: async () => unwrapResponse<GamePackageList>(await gamePackagesApi.list()),
  })
}

export function useGamePackage(id?: string) {
  return useQuery<GamePackageDetail>({
    queryKey: ['game-packages', id],
    queryFn: async () => {
      if (!id) throw new Error('game package id is required')
      return unwrapResponse<GamePackageDetail>(await gamePackagesApi.detail(id))
    },
    enabled: Boolean(id),
  })
}

export function useInstallGameAcceptanceFixtures() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => unwrapResponse(await gamePackagesApi.installAcceptanceFixtures()),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-packages'] }) },
  })
}

export function useSetGamePackageEnabled() {
  const queryClient = useQueryClient()
  return useMutation<unknown, Error, { id: string; enabled: boolean }>({
    mutationFn: async ({ id, enabled }) => unwrapResponse(await gamePackagesApi.setEnabled(id, enabled)),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['game-packages'] }) },
  })
}

export function useGamePackageTestRun() {
  return useMutation<GameTestRunResult, Error, { id: string; commands: GameCommand[]; initialState?: Record<string, unknown> }>({
    mutationFn: async ({ id, commands, initialState }) =>
      unwrapResponse<GameTestRunResult>(await gamePackagesApi.testRun(id, commands, initialState)),
  })
}
