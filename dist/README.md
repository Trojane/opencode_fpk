# 成品包

| 项 | 值 |
|---|---|
| 文件名 | `opencode_all_2.2.5.fpk` |
| 大小 | 246,617,275 字节（236 MB） |
| SHA-256 | `73efb29af0dd3141b0002884cadf6a8c3188a15964faac9aa87b2d0c7426d6d7` |
| appname / version | `opencode` / `2.2.5` |
| platform | `all`（x86_64 + arm64） |

## 为什么要分卷

Gitee 免费仓库单文件上限 100 MB，成品 236 MB，已按 80 MB 切成 3 卷：

```
opencode_all_2.2.5.fpk.part-00   80 MB
opencode_all_2.2.5.fpk.part-01   80 MB
opencode_all_2.2.5.fpk.part-02   76 MB
```

## 合并

Linux / macOS：

```bash
cd dist
chmod +x join.sh
./join.sh
```

或手动：

```bash
cat opencode_all_2.2.5.fpk.part-* > opencode_all_2.2.5.fpk
sha256sum opencode_all_2.2.5.fpk
```

Windows CMD：

```bat
copy /b opencode_all_2.2.5.fpk.part-00+opencode_all_2.2.5.fpk.part-01+opencode_all_2.2.5.fpk.part-02 opencode_all_2.2.5.fpk
certutil -hashfile opencode_all_2.2.5.fpk SHA256
```

合并后校验值必须是：

```
73efb29af0dd3141b0002884cadf6a8c3188a15964faac9aa87b2d0c7426d6d7
```

不一致说明下载损坏，请重新下载对应分卷。各分卷校验值见 `parts.sha256`。

## 安装

飞牛 → 应用中心 → 手动安装 → 选合并后的 `opencode_all_2.2.5.fpk`。
