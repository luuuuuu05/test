# MR Flag Battle — 开发者接手指南

> Unity 2022.3.62f3c1 | Pico 4 Ultra Enterprise | Go 后端

---

## 第一步：打开项目

1. 用 **Unity Hub** 打开 `C:\Users\HP\Documents\1\Flag` 文件夹
2. 确保 Unity 版本是 **2022.3.62f3c1**，不一致时用 Unity Hub 下载对应版本
3. 首次打开需要较长时间导入资源，等进度条走完

## 第二步：启动 Pico MCP 桥接（开发用）

1. 菜单栏 → `Window → MCP for Unity`
2. 点 **Auto-Setup**  → 点 **Start Bridge**
3. 如果报 "Session not found"，关了重开

## 第三步：构建部署到 Pico

1. 菜单栏 → `File → Build Settings`
2. 确认 **平台 = Android**，场景列表有 5 个场景（MainMenu, MapSelect, MRGame, MapEditor, MultiGame）
3. 点 **Build And Run**，Pico 通过 USB 连接电脑
4. 第一次构建需要 1-5 分钟（IL2CPP 编译），后续增量构建快很多

## 第四步：启动 Go 后端（联机需要）

```powershell
cd 后端目录
go run ./cmd/server -config config.yaml
```

后端默认端口：HTTP 8080 / WebSocket 8081 / UDP 9090

## 第五步：场景结构速览

打开 `Assets/Scenes/` 目录，5 个场景的职责：

| 场景 | 干什么 | 入口 |
|------|--------|------|
| **MainMenu** | 主菜单，选 Host/Join/Solo | 启动场景 |
| **MapSelect** | 选地图 + 跳地图编辑 | Host/Join 后进入 |
| **MapEditor** | 地图编辑器，手柄搭建场景 | MapSelect 点 +New Map |
| **MRGame** | 单人夺旗游戏（30s 倒计时） | Solo→选地图 |
| **MultiGame** | 双人联机夺旗 | Host/Join→选地图 |

## 第六步：模型替换

### 旗子模型

目前是圆柱体占位。替换为你的 3D 模型：

1. 打开 `Assets/Prefabs/MapObjects/Flag_Gold.prefab`
2. 删掉里面的 Cylinder
3. 把你的金旗子 3D 模型（.fbx/.obj）拖入 Prefab
4. 同样操作 `Flag_Red.prefab` 和 `Flag_White.prefab`

### 场景搭建物体（墙壁、箱子等）

- `Assets/Prefabs/Wall.prefab` → 替换为你的墙壁模型
- `Assets/Prefabs/Box.prefab` → 替换为你的箱子模型
- `Assets/Prefabs/Pillar.prefab` → 替换为你的柱子模型
- `Assets/Prefabs/Barrel.prefab` → 替换为你的桶模型
- `Assets/Prefabs/Ramp.prefab` → 替换为你的坡道模型
- `Assets/Prefabs/Sphere.prefab` → 替换为你的球模型

**替换后必须操作**：`Window → Asset Management → Addressables → Build`

### 玩家模型

- 自己的 Avatar：选中 MultiGame 场景 → 找到 `GameRoot` → `LocalPlayer` 组件 → 把人物 Prefab 拖到 `Avatar Prefab` 槽
- 对手的 Avatar：选中 MultiGame 场景 → 找到 `GameRoot` → `RemotePlayerManager` 组件 → 把人物 Prefab 拖到 `Remote Player Prefab` 槽

## 第七步：修改游戏参数

### 单机模式（MRGame）

打开 `MRGameManager.cs`，可以改：

```csharp
public float gameDuration = 30f;   // 游戏时长（秒）
public float respawnDelay = 3f;    // 旗子重生间隔
```

`SpawnFlags()` 方法里可以改旗子类型、分值、颜色、位置。

### 联机模式（MultiGame）

打开 `MultiGameManager.cs`，同样改 `_gameDuration`。

旗子由后端控制（`s_flag_spawn` 消息），修改后端配置即可。

## 第八步：加新功能

### 加音效

在 `MRFlag.cs` 的 `OnGrabbed()` 方法最后加：

```csharp
GetComponent<AudioSource>()?.Play();
```

### 加粒子特效

```csharp
// 抓旗时播放爆炸特效
Instantiate(explosionPrefab, transform.position, Quaternion.identity);
```

### 加地图编辑物体

在 `MapEditorManager.cs` 的 `placeables` 列表里（Inspector 中能看到）增加新条目，填 `id`, `displayName`, `prefabRef`（Addressable Prefab 引用）。

## 第九步：⚠️ 注意事项

### 🚫 绝对不能做的事

1. **不要碰 XR Rig 的子对象** — Pico Building Block 创建的，改了就废
2. **不要改网络层文件** — `NetworkManager.cs`, `WSClient.cs`, `UDPClient.cs`, `MsgPack.cs`, `Messages.cs`, `RestClient.cs` — 这些和后端协议强绑定
3. **GameContext 不能和别的脚本挂同一个 GameObject** — 必须独立
4. **不要用 `SceneManager.LoadSceneAsync`** — 只能用同步 `LoadScene`

### ✅ 常见操作

| 想做的事 | 去哪里改 |
|----------|----------|
| 改分数/时间 | `MRGameManager.cs` / `MultiGameManager.cs` |
| 换旗子模型 | `Assets/Prefabs/MapObjects/` 下的 Prefab |
| 改 UI 文字/颜色 | 打开对应场景，选 `UI_Root` 下的文字组件 |
| 改主菜单按钮文字 | 打开 MainMenu 场景，找到按钮下的 Text 对象 |
| 加特效应 | `MRFlag.cs` `OnGrabbed()` |
| 改多人游戏逻辑 | `MultiGameManager.cs` `HandleServerMessage()` |
| 改地图存储格式 | `LocalMapStore.cs` |

### 联机调试

1. 电脑开 Go 后端
2. 两台 Pico 连同一局域网
3. MainMenu 点 IP 按钮选 `10.19.89.160`（或输入后端 IP）
4. 一台点 **Host Game**，一台点 **Join Game**
5. 都进入 MapSelect → Host 选地图 → 两人进 MultiGame → 点 Start Game

### 读 Pico 日志

```powershell
adb logcat -s Unity
```
