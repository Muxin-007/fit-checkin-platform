# 再鸽一天后端

完整的运行、微信接入与部署说明见仓库根目录 [README.md](../README.md)。

## API 前缀

```text
/api/fitness
```

主要路由分为：

- `/auth`：微信登录与会话退出
- `/profile`：用户资料、偏好和注销
- `/groups`、`/invitations`：小组、成员和邀请
- `/checkins`：打卡、详情与日历
- `/subscriptions`、`/reminders`：订阅授权和提醒
- `/reports`：用户举报
- `/security/media-callback`：微信图片内容安全回调
- `/admin`：运营接口
- `/ops/`：嵌入式轻量运营台

服务启动时由 Ent 创建当前业务 schema。生产密钥通过 `platform.yaml` 中的 `${FITNESS_*}` 环境变量展开，不应写入仓库。
