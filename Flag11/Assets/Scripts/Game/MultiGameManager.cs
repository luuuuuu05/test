using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;
using UnityEngine.SceneManagement;
using Newtonsoft.Json.Linq;

public enum MultiState { Idle, Waiting, Countdown, Playing, Ended }

public class MultiGameManager : MonoBehaviour
{
    public MultiState State { get; private set; } = MultiState.Idle;
    private NetworkManager _net;
    private FlagManager _flagMgr;

    [Header("Map Objects")]
    public SceneObjectRegistry objectRegistry;

    [Header("Game")]
    public int myScore, opponentScore;
    public string opponentID;
    public bool myDoubleActive, opponentDoubleActive;

    [Header("UI - Waiting")]
    public GameObject waitPanel;
    public Text playerCountText;
    public Button startGameBtn;
    public Button backBtn;

    [Header("UI - Game")]
    public GameObject countdownPanel;
    public Text countdownText, scoreText, timerText;
    public GameObject gameOverPanel;
    public Text finalScoreText;

    private float _gameStartTime;
    private float _gameDuration = 30f;

    void Start()
    {
        _net = NetworkManager.Instance;
        _flagMgr = GetComponent<FlagManager>();

        _net.OnServerMessage += HandleServerMessage;
        _net.OnConnected += () => { if (_net != null) _net.StartUdp(GameContext.Instance.serverIP, GameContext.Instance.udpPort); _net.StartHeartbeat(); };

        if (startGameBtn != null) startGameBtn.onClick.AddListener(StartGame);
        startGameBtn?.gameObject.SetActive(true);

        WireBackButton();
        if (backBtn != null) backBtn.onClick.AddListener(() => SceneManager.LoadScene(0));
        LoadSelectedMap();

        waitPanel?.SetActive(true);
        countdownPanel?.SetActive(false);
        gameOverPanel?.SetActive(false);

        // Show current player count from server
        if (playerCountText != null)
            playerCountText.text = (NetworkManager.Instance.IsConnected ? "?" : "") + "/2";
    }

    void WireBackButton()
    {
        var canvas = GameObject.Find("UI_Root");
        if (canvas == null) return;
        var backBtn = canvas.transform.Find("BackBtn");
        if (backBtn == null) return;
        var btn = backBtn.GetComponent<Button>();
        btn.onClick.RemoveAllListeners();
        btn.onClick.AddListener(() => SceneManager.LoadScene(0));
    }

    void LoadSelectedMap()
    {
        MapLoaderHelper.LoadMapIntoScene(objectRegistry);
    }

    void Update()
    {
        if (State == MultiState.Playing && timerText != null)
            timerText.text = Mathf.Max(0, (int)(_gameDuration - (Time.time - _gameStartTime))) + "s";
    }

    void HandleServerMessage(string type, JObject payload)
    {
        switch (type)
        {
            case MessageTypes.S_PLAYER_JOINED:
                var pj = payload.ToObject<PlayerJoinedPayload>();
                if (pj.player_id != GameContext.Instance.PlayerID)
                    opponentID = pj.player_id;
                if (playerCountText != null) playerCountText.text = pj.player_count + "/2";
                if (pj.player_count >= 2) startGameBtn?.gameObject.SetActive(true);
                break;

            case MessageTypes.S_GAME_COUNTDOWN:
                StartCoroutine(ShowCountdown(payload.ToObject<CountdownPayload>().count));
                break;

            case MessageTypes.S_GAME_START:
                State = MultiState.Playing;
                _gameStartTime = Time.time;
                if (payload.TryGetValue("duration", out var dur))
                    _gameDuration = dur.Value<float>();
                waitPanel?.SetActive(false);
                countdownPanel?.SetActive(false);
                break;

            case MessageTypes.S_FLAG_SPAWN:
                _flagMgr?.SpawnFlag(payload.ToObject<FlagSpawnPayload>());
                break;

            case MessageTypes.S_FLAG_REMOVE:
                _flagMgr?.RemoveFlag(payload.ToObject<FlagRemovePayload>().flag_id);
                break;

            case MessageTypes.S_SCORE_UPDATE:
                var su = payload.ToObject<ScoreUpdatePayload>();
                if (su.player_id == GameContext.Instance.PlayerID) myScore = su.total;
                else opponentScore = su.total;
                if (scoreText != null)
                {
                    string myExtra = myDoubleActive ? " x2" : "";
                    string opExtra = opponentDoubleActive ? " x2" : "";
                    scoreText.text = "You:" + myScore + myExtra + " | Opp:" + opponentScore + opExtra;
                }
                break;

            case MessageTypes.S_GAME_END:
                State = MultiState.Ended;
                var ge = payload.ToObject<GameEndPayload>();
                gameOverPanel?.SetActive(true);
                if (finalScoreText != null)
                    finalScoreText.text = "You:" + myScore + " | Opp:" + opponentScore;
                break;

            case MessageTypes.S_BUFF_START:
                {
                    var buf = payload.ToObject<BuffPayload>();
                    if (buf.player_id == GameContext.Instance.PlayerID)
                        myDoubleActive = true;
                    else
                        opponentDoubleActive = true;
                }
                break;

            case MessageTypes.S_BUFF_END:
                {
                    var bufEnd = payload.ToObject<BuffEndPayload>();
                    if (bufEnd.player_id == GameContext.Instance.PlayerID)
                        myDoubleActive = false;
                    else
                        opponentDoubleActive = false;
                }
                break;

            case MessageTypes.S_ERROR:
                Debug.LogError("[Multi] " + payload.ToObject<ErrorPayload>().message);
                break;

            case "s_scene_map_saved":
            case "s_scene_map_snapshot":
            case "s_scene_map_update":
                string mi = payload["map_id"]?.ToString() ?? "default";
                var mo = payload["map"] as JObject;
                if (mo != null)
                {
                    var objs = mo["objects"];
                    if (objs != null)
                    {
                        string sj = "{\"mapName\":\"" + mi + "\",\"savedAt\":0,\"objectCount\":" + (objs as JArray)?.Count + ",\"objects\":" + objs.ToString() + "}";
                        string d = Application.persistentDataPath + "/maps/";
                        if (!System.IO.Directory.Exists(d)) System.IO.Directory.CreateDirectory(d);
                        System.IO.File.WriteAllText(d + mi + ".json", sj);
                    }
                }
                PlayerPrefs.SetString("SelectedMap", mi);
                LoadSelectedMap();
                break;
        }
    }

    IEnumerator ShowCountdown(int count)
    {
        State = MultiState.Countdown;
        countdownPanel?.SetActive(true);
        for (int i = count; i >= 0; i--)
        {
            if (countdownText != null) countdownText.text = i > 0 ? i.ToString() : "GO!";
            yield return new WaitForSeconds(1f);
        }
    }

    public void StartGame()
    {
        _net.SendToServer("c_start_game", null);
    }

    public void OnFlagGrabbed(FlagObject flag)
    {
        if (State != MultiState.Playing) return;
        flag.gameObject.SetActive(false); // immediate local feedback
        _net.GrabFlag(flag.FlagID, flag.transform.position);
    }

    void OnDestroy() { if (_net != null) _net.OnServerMessage -= HandleServerMessage; }
}
