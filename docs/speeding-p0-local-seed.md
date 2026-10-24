# Speeding P0 本地导入

先确保 Hub 已至少启动过一次（数据库表已建好），且当前租户已有 `director`、`cfo`、`counsel` 三个 Agent。先预览：

```sh
go run ./cmd/seed-speeding-p0 --dry-run
```

确认后导入：

```sh
go run ./cmd/seed-speeding-p0
```

命令使用当前 `DATABASE_URL`，默认租户为 `default`。Agent 名称不同时可传入 `--director`、`--finance`、`--legal` 和 `--board-rival`。缺少的 board rival 会通过 Agent 服务创建；其他三个缺失时命令会安全停止。

导入固定创建 key 为 `speeding-p0-local` 的独立草稿。如已存在，再次执行只报告 `alreadyExists`，不更新、不覆盖。输出中 `pendingAssets` 是待上传素材清单；原 JSON 里的 `asset://` 槽位不会当作有效资源写入草稿。
