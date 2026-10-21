# OpenAI 兼容视频接口

支持 OpenAI 平台类型的 API Key 上游，包括 A6API 的 `openai-video` 模型。

| 操作 | 路径 |
| --- | --- |
| 提交生成任务 | `POST /v1/videos` |
| 提交兼容别名 | `POST /v1/video/generations` |
| 查询任务 | `GET /v1/videos/{id}` |
| 获取视频内容 | `GET /v1/videos/{id}/content` |

以上路径同时提供去掉 `/v1` 的别名。提交别名统一转发到上游 `POST /v1/videos`。
Grok 分组原有的 `/v1/videos/generations`、编辑、延长和任务查询接口继续使用 Grok 协议。

## 管理配置

1. 上游账户选择 OpenAI / API Key，Base URL 填上游根地址或 `/v1` 地址，启用“视频生成”端点能力。显式配置过旧能力列表的账户，需要手动勾选视频能力；未限制能力的账户默认支持。
2. 将账户加入 OpenAI 分组，启用该分组已有的媒体生成权限（`allow_image_generation`）。
3. 在分组的视频计费设置中填写对应分辨率的 USD/秒价格，或配置渠道模型定价。非 Grok 视频没有可靠的内置价目表：缺少定价时，提交会在调用上游前返回 `video_pricing_not_configured`。填写 0 表示明确免费；简易模式跳过定价检查。
4. 调用端必须选择视频协议。聊天客户端追加 `/chat/completions` 的行为不会因为模型名或 Base URL 改变而切换协议。

JSON、multipart/form-data 均可提交；模型映射适用于两种格式，multipart 的 `input_reference` 文件和其他供应商参数保留。建议显式指定 `seconds`（或上游支持的 `duration`），上游返回的秒数优先用于用量记录。大小、声音、参考图等能力和参数范围由具体上游模型决定。

## 账号测试入口

在账号列表打开“测试”，选择模型及“视频生成”测试模式，填写提示词、时长和尺寸后提交。仅启用视频能力的账号默认选择视频模式；同时支持聊天的账号需手动选择。

测试直接向当前上游账号提交一次 `POST /v1/videos`，会产生上游费用，不走用户余额计费。成功表示任务已受理，界面显示任务 ID 和上游状态；管理员可点击“刷新状态 / 获取视频”查询原任务，任务完成后在测试弹窗内播放或下载。查询和下载使用原上游账号，不创建新任务，也不扣中转用户余额。测试任务不绑定普通中转 API Key。请求中断后先检查原任务，避免重复付费；测试不会自动重试，后台定时测试也不会自动提交视频任务。

最近一次任务按账号保存在当前浏览器标签页的 `sessionStorage` 中，关闭弹窗或刷新页面后可恢复。升级前的任务可以填写任务 ID，点击“查询已有任务”取回结果；必须选择原提交账号。回放需要先完整下载视频到浏览器，后台服务器按流转发。开发规格与验收记录见 [管理员视频测试回放](ADMIN_VIDEO_TEST_PLAYBACK.md)。

## 调用示例

下列 `RELAY_BASE_URL` 指向自己的中转，`RELAY_API_KEY` 使用中转密钥，不使用供应商密钥。

```bash
curl "$RELAY_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $RELAY_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"model":"minimax-h3","prompt":"A paper boat floating on a calm pond.","seconds":"6","size":"1280x720"}'

curl "$RELAY_BASE_URL/v1/videos/$VIDEO_TASK_ID" \
  -H "Authorization: Bearer $RELAY_API_KEY"

curl "$RELAY_BASE_URL/v1/videos/$VIDEO_TASK_ID/content" \
  -H "Authorization: Bearer $RELAY_API_KEY" \
  -o video.mp4
```

任务响应、状态及错误正文保持上游格式；内容接口支持 `variant` 查询参数、Range 请求和流式下载。

## 任务归属与计费

任务 ID 与提交账户的绑定保存在共享 Redis 中，按中转 API Key ID 和分组隔离，保留 7 天。
查询和下载固定使用原账户，不会因为账户负载或模型调度切换到其他供应商。
另一把中转密钥或未知任务返回 404；Redis 故障返回 503。需保留 Redis 数据，清空 Redis 后不能自动找回绑定。
上游接受任务后若绑定写入失败，中转保留成功响应和任务 ID，并记录服务端错误日志；后续查询/下载需要恢复该任务绑定。

沿用现有媒体任务的提交计费方式：上游成功接受任务后记录一次用量，查询和下载不重复计费；异步失败不自动退款。
Grok 模型保留原有时长限制，其他模型的计费时长不会被截断为 15 秒。
视频用量在异步记录队列溢出时同步兜底，避免直接丢弃计费记录。

为避免重复付费，提交发生网络错误或上游错误时不自动切换账户重试；客户端提供的 `Idempotency-Key` 会透传给上游，是否支持幂等由供应商决定。

## 验证

本地回归覆盖 JSON/multipart、模型映射、路由、任务账户绑定、跨密钥隔离、上游错误、内容下载、计费和前端能力设置。

真实测试为显式启用，默认测试不会调用付费上游：

```bash
# 将供应商密钥放入环境变量 A6API_VIDEO_TEST_KEY。
# 此测试创建一次 6 秒 720p 的 minimax-h3 视频任务，会产生上游费用。
go test ./internal/service -run '^TestOpenAIVideosA6Live$' -count=1 -v -timeout=22m
```

`A6API_VIDEO_TEST_OUTPUT` 可指定响应和 MP4 的保存目录；中断后可设置 `A6API_VIDEO_TASK_ID` 继续查询同一任务，避免再次提交。
