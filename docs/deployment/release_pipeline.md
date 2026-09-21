# Release Pipeline

本仓库的正式发布入口是 GitHub Actions 的 `Release` workflow。它只在推送
`v*` tag 时创建 GitHub Release；手动运行 workflow 只生成供维护者检查的
Preview artifacts，不会创建正式 Release。

## 发布前准备

正式 macOS Release 需要 Apple Developer ID 证书和 notarization 凭据。将以下
内容配置到仓库的 GitHub Actions Secrets：

| Secret | 内容 |
| --- | --- |
| `APPLE_CERTIFICATE_P12_BASE64` | Developer ID Application 证书导出的 `.p12` 文件，经 base64 编码 |
| `APPLE_CERTIFICATE_PASSWORD` | `.p12` 导出密码 |
| `APPLE_SIGNING_IDENTITY` | 证书完整名称，例如 `Developer ID Application: Example, Inc. (TEAMID)` |
| `APPLE_ID` | Apple Developer 账号邮箱 |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `APPLE_APP_SPECIFIC_PASSWORD` | 用于 `notarytool` 的 Apple ID app-specific password |

证书和密码不进入仓库、不写入 workflow 文件。tag 发布时如果这些 Secrets
缺失，macOS job 会明确失败，整个 Release 不会被发布成“看似成功但未公证”的
状态。手动运行 workflow 可以不配置这些 Secrets，用于验证其他平台和未签名的
macOS Preview artifact。

## 发布命令

先确保目标 commit 已通过 CI，再创建并推送版本 tag：

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

## 自动产物

Release workflow 会并行生成：

```text
go-e2e-v0.1.0-macos-arm64.dmg
go-e2e-v0.1.0-macos-amd64.dmg
go-e2e_v0.1.0_linux_amd64.tar.gz
go-e2e_v0.1.0_windows_amd64.zip
go-e2e-setup.exe
SHA256SUMS
RELEASE_NOTES.md
```

CLI 归档也会继续保留在同一 Release 中。所有最终资产由 publish job 统一生成
`SHA256SUMS`，避免不同构建 job 产生互相覆盖的校验文件。

## 验收

`publish` 成功后，workflow 会从刚创建的 GitHub Release 重新下载资产并自动执行
三平台验收；维护者仍需查看 job 日志和 Release 页面：

1. macOS DMG 挂载、安装复制、签名/notarization 校验和启动是否通过。
2. Windows 安装程序是否能安装、启动和卸载，并保留用户数据。
3. Linux 归档是否能解包，`go-e2e-desktop` 与旁边的 `go-e2e` 是否都存在，sidecar 版本是否正确。
4. 各平台下载的资产校验和是否通过。
5. Release 页面、README 安装说明和版本 tag 是否指向同一个 commit。

移动端 Android/iOS 不属于当前桌面 Release workflow，需要后续分别建立
Android APK/AAB 和 iOS IPA/TestFlight 发布链路。
