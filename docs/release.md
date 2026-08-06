# 发布与服务器部署

## 创建发行版

1. 确认 `go test ./...` 和跨平台构建通过。
2. 创建语义化标签，例如 `setup-v1.4.0`。
3. 推送标签，等待 `Release AiEngine Setup` 工作流完成。
4. 检查 GitHub Release 包含六个平台压缩包、两个引导脚本、`CHECKSUMS.txt` 和 `latest.json`。

```sh
git tag -a setup-v1.4.0 -m "AiEngine Setup 1.4.0"
git push origin setup-v1.4.0
gh run watch
```

工作流也支持两种恢复入口：

- 在 GitHub Actions 页面手动运行 `Release AiEngine Setup`，输入已经存在的 `setup-v*` 标签。
- 推送同版本的 `release-v*` 分支。例如 `release-v1.4.2` 会创建 `setup-v1.4.2` Release；对应标签必须已经存在。

```sh
git push origin HEAD:release-v1.4.2
```

这些入口只用于恢复未被调度或被取消的发布任务，不需要重新修改安装器代码。

## 发布 GitHub 备用资产

若 Actions 未调度或 GitHub Release 暂时不可用，可通过 SSH 直接更新 `setup-assets` 分支：

```sh
./deploy/publish-assets-branch.sh setup-v1.4.5
```

脚本会在隔离的临时工作树中检出指定标签，构建并校验六个平台包，然后只将当前版本资产推送到 `setup-assets` 分支。在线安装器会在 AiEngine 主下载源不可用时自动使用该分支；此路径不需要 GitHub API Token。

## 部署当前版本

在 NewAPI 所在服务器的仓库目录运行：

```sh
sudo ./deploy/publish.sh setup-v1.4.0
```

固定路径：

- 发布资产：`/www/wwwroot/newapi.aiare.cloud/aiengine-setup/releases/<tag>`
- 当前版本：`/www/wwwroot/newapi.aiare.cloud/aiengine-setup/current`
- Nginx 扩展：`/www/server/panel/vhost/nginx/extension/newapi.aiare.cloud/aiengine-setup.conf`

部署不会修改 NewAPI 容器、数据库、API 路由或现有站点配置，只会安装独立静态下载 location。`current` 不缓存，带版本的发布目录长期缓存。

## 验证

```sh
curl -fsS https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex --dry-run
curl -fsS https://modelapi.aiaiaiaiai.cloud/aiengine-setup/current/latest.json
curl -fsS https://modelapi.aiaiaiaiai.cloud/aiengine-setup/current/CHECKSUMS.txt
```

`--dry-run` 只在执行端检测已有 CLI，不读取密钥、不修改文件。完整 API 验证需要客户自己的 AiEngine API 密钥。

## 回滚

重新部署上一个已发布标签即可：

```sh
sudo ./deploy/publish.sh setup-v1.1.0
```

发布目录不会被覆盖或自动删除。若目标标签已部署过，可将 `current` 原子切换到对应 `releases/<tag>`，再执行 `nginx -t && nginx -s reload`。
