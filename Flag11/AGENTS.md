# AGENTS.md — Flag (Unity MR Project)

## Quick Facts
- **Engine:** Unity 2022.3.62f3c1 (LTS)
- **Target:** Pico 4 Ultra Enterprise (Android)
- **Project:** MR Flag Battle — 2-player mixed-reality flag-capture game
- **Server:** External Go backend (separate repo) — ports: HTTP 8080, WS 8081, UDP 9090
- **No tests, CI, or CLI build** — run via Unity Editor only
- **JSON:** Newtonsoft.Json **3.2.1** (via Pico SDK dependency, not in manifest.json directly)

## XR Rig Rules (must obey)

**Never build XR Rig manually.** Create via Pico Building Block menu:
1. `PICO/PICO Building Blocks/PICO Controller/PICO Controller Tracking`
2. `PICO/PICO Building Blocks/PICO Hand/PICO Hand Tracking`
3. `PICO/PICO Building Blocks/PICO Video Seethrough/PICO Video Seethrough`
4. Set `XROrigin.RequestedTrackingOriginMode = Floor` and `CameraYOffset = 0`

- **Do not add/remove components** on XR Rig children (`[Building Block]` objects, XR Interaction Manager, EventSystem).
- All game objects (ground, flags, GameManager, etc.) must be **scene root objects** — never parent under XR Origin or Camera.

## Architecture

- **Unity client only.** Scripts under `Assets/Scripts/` grouped: Protocol, Network, Game, Player, Flag, UI, MR.
- `GameContext` is a DontDestroyOnLoad singleton holding runtime state. **Must be on its own GameObject** — never share with GameManager or others (Awake Destroy cascade risk).
- Solo vs multiplayer decided at runtime: `GameContext.serverIP` empty = solo, set = multiplayer.
- `SceneObjectRegistry.asset` (`Assets/Prefabs/`) maps prefab names to GameObjects for map loading. Entries must match the `PlaceableDef.id` / `SceneObject.prefab` strings stored in map JSON.

## Scene Flow

Build scene order (index 0 = MainMenu, used by `LoadScene(0)` calls):
1. **MainMenu** — Host / Join / Solo entry
2. **MapSelect** — pick map or create new
3. **MRGame** — single-player 30s flag capture (`MRGameManager` + `MRFlag`)
4. **MapEditor** — XR controller place/select/edit objects
5. **MultiGame** — 2-player networked game (`MultiGameManager` + `FlagManager` + `RemotePlayerManager`)

`SampleScene.unity` exists on disk but is **not in the build** — ignore it.

## Pico XR SDK
- **Pico Integration SDK 3.4.0** embedded at `Packages/PICO Unity Integration SDK-3.4.0-20260226/` — do not move or rename.
- MR centralized in `PicoMRManager.cs`: VST enable, SenseDataProvider, shared anchor create/persist/upload, drift correction in FixedUpdate.
- `PicoMRManager.Awake()` sets `RenderSettings.skybox = null`, camera clearFlags = SolidColor with transparent black.
- `CanvasXRSetup.cs`: `ScreenSpaceCamera` on Android (headset), `ScreenSpaceOverlay` in Editor.
- Dependencies: XR Management 4.4.0, XR Interaction Toolkit 2.6.4, XR Core Utils 2.5.2.

## Networking Protocol (MultiGame)
- **WS:** `System.Net.WebSockets.ClientWebSocket` — no external WS library. JSON messages via Newtonsoft.Json.
- **UDP:** `MsgPack.cs` custom encoder/decoder for 10-field position packets. Format must match Go backend exactly.
- **REST:** `RestClient.cs` (UnityWebRequest) for room CRUD on `:8080`.
- **Position sync:** `LocalPlayer.Update()` sends UDP every frame (no client-side throttle — server handles rate).

## MCP for Unity
- Package: `com.coplaydev.unity-mcp` via git URL in manifest.
- Config: `opencode.json` → `http://127.0.0.1:8080/mcp`.
- Editor: `Window > MCP for Unity` → **Auto-Setup** → **Start Bridge**.
- If "Session not found", restart the bridge.

## DO NOT EDIT

| File | Reason |
|------|--------|
| `NetworkManager.cs` | Global singleton, cross-scene. Awake/Update/message queue stable |
| `WSClient.cs` | System.Net.WebSockets async send/recv |
| `UDPClient.cs` | MsgPack encode/decode, 30Hz position sync |
| `MsgPack.cs` | Custom MessagePack serializer, format matches Go backend exactly |
| `Messages.cs` | All protocol message types, 1:1 with backend |
| `RestClient.cs` | HTTP REST client for room operations |
| `GameContext.cs` | Global state singleton, must stay on independent GameObject |
| XR Rig `[Building Block]` objects | Pico BB created, components frozen |

## Safe to Edit

| File | What to change | Constraint |
|------|----------------|------------|
| `MRGameManager.cs` | Duration, scores, flag count, respawn | Keep `public` method signatures |
| `MultiGameManager.cs` | Same + network message handling | Don't change `HandleServerMessage` dispatch types |
| `MRFlag.cs` | Visuals, effects, grab feedback | `Setup()` signature is the entry point |
| `FlagObject.cs` | Prefab slots, grab callbacks | Assign goldPrefab/redPrefab/whitePrefab in Inspector |
| `MapEditorManager.cs` | Placeable list, edit panel UI | Keep `PlaceableDef` structure |
| `MapSelectManager.cs` | Map list UI, delete confirm | Keep `SelectMap()` signature |
| `JoinPanel.cs` | MainMenu UI layout | Keep `OnHost/OnJoin/OnSolo` entry points |
| `LocalPlayer.cs` / `RemotePlayer.cs` | Avatar model, effects, interpolation | |
| `FlagManager.cs` | Spawn/remove logic | |
| `LocalMapStore.cs` | Map storage | Keep JSON format compatible |

## Known Constraints

- **Scene loading must be synchronous** — `SceneManager.LoadScene()` only, no `LoadSceneAsync` (initialization timing issues).
- **Pico device must disable `StandaloneInputModule`** — each scene's EventSystem should only have `XRUIInputModule`.
- **Addressables rebuild required** after modifying Prefabs: `Window > Asset Management > Addressables > Build`.
- **Netick package** (`com.karrar.netick`) and prefabs (`NetickFlag`, `NetickPlayer`, `NetickSandbox`) are **unused legacy** — do not build on them.
- **MainMenu is build index 0** — multiple scripts use `LoadScene(0)` to return to main menu.
- **No keyboard input** in any script — all interaction is XR controller / hand tracking based.
