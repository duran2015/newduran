# Agent Guidelines

## Frontend Conventions

### Product-facing pages must use human language

Pages used by product managers, operators, creators, or players must explain the
user goal, available action, current state, and outcome in plain language. Do
not expose implementation terms such as Schema, Manifest, Payload, Runtime,
callback, raw event names, database IDs, or internal field keys in the default
view. Keep the internal names in code and APIs; when operators genuinely need
them, place them in a clearly labelled, collapsed **Developer information**
section. Empty states, errors, buttons, and audit trails must tell the user what
happened and what to do next instead of returning internal status strings.

### Use PrimaryButton for all primary-filled buttons

All primary-filled buttons (`type="primary"` Buttons) must use the shared `frontend/src/components/PrimaryButton` component instead of writing `<Button type="primary">` directly, to keep the primary button style consistent across pages (accent fill, unified disabled state).

```tsx
// ✅ Correct
import PrimaryButton from '@/components/PrimaryButton'
<PrimaryButton icon={<PlusIcon size={16} />}>New</PrimaryButton>

// ❌ Wrong
<Button type="primary">New</Button>
```

Modal OK buttons don't need to be replaced wholesale — inject the style via `okButtonProps` with `usePrimaryButtonStyle()` instead (see the comment at the top of `frontend/src/components/PrimaryButton.tsx` for details).

## Release Process

1. Before releasing: confirm CI + CodeQL on the latest `main` are green, and run local pre-checks (`gofmt -l`, `go vet`).
2. Tag and push `hub-vX.Y.Z` on `main` — this triggers `release.yml`, which pushes the image to both `swr.cn-east-3.myhuaweicloud.com/zerone/agent-hub` and `docker.io/zeroneai/agent-hub` (`vX.Y.Z` + `latest`).
3. After the workflow succeeds, manually create the GitHub Release with `gh release create`: title `Hub-vX.Y.Z`, marked as Latest, notes in Chinese following the structure of previous releases (theme section / 变更 with PR numbers / 镜像 / 验证 / compare link).

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues (zerone-agents/agent-hub), accessed via the `gh` CLI. See `docs/agents/issue-tracker.md`.
