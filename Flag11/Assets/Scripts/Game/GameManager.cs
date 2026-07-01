using System.Collections.Generic;
using UnityEngine;
using Newtonsoft.Json.Linq;

public class GameManager : MonoBehaviour
{
    public GameStateMachine State { get; private set; } = new();

    private NetworkManager _network;
    private FlagManager _flagManager;

    public string opponentPlayerID;
    public int myScore;
    public int opponentScore;
    public bool myDoubleActive;
    public bool opponentDoubleActive;
    public List<PlayerScoreEntry> scoreboard = new();

    public event System.Action<int> OnCountdown;
    public event System.Action OnGameStarted;
    public event System.Action<GameEndPayload> OnGameEnded;
    public event System.Action<ScoreUpdatePayload> OnScoreUpdated;
    public event System.Action<PlayerJoinedPayload> OnPlayerJoined;

    void Start()
    {
        _network = GetComponent<NetworkManager>();
        if (_network == null) _network = gameObject.AddComponent<NetworkManager>();
        _flagManager = GetComponent<FlagManager>();
        if (_flagManager == null) _flagManager = gameObject.AddComponent<FlagManager>();

        _network.OnServerMessage += HandleServerMessage;
        _network.OnConnected += () => State.TransitionTo(GameState.Waiting);
        _network.OnDisconnected += _ => State.TransitionTo(GameState.Idle);
    }

    public void OnFlagGrabbed(FlagObject flag)
    {
        if (State.CurrentState != GameState.Playing) return;
        Debug.Log($"[Game] Flag grabbed: {flag.FlagID} {flag.FlagType} = {flag.Score}pts");
        myScore += flag.Score;
        _flagManager.RemoveFlag(flag.FlagID);

        OnScoreUpdated?.Invoke(new ScoreUpdatePayload
        {
            player_id = GameContext.Instance?.PlayerID ?? "P1",
            delta = flag.Score, total = myScore,
            scoreboard = new List<PlayerScoreEntry> { new PlayerScoreEntry { player_id = GameContext.Instance?.PlayerID ?? "P1", display_name = "You", score = myScore } }
        });

        // Respawn after delay
        StartCoroutine(RespawnFlag(flag));
    }

    private System.Collections.IEnumerator RespawnFlag(FlagObject old)
    {
        yield return new WaitForSeconds(3f);
        if (State.CurrentState != GameState.Playing) yield break;

        Vector3 pos = new Vector3(Random.Range(-3f, 3f), 0.3f, Random.Range(0.5f, 3f));
        string[] types = { "gold", "red", "white", "white" };
        string t = types[Random.Range(0, types.Length)];
        int sc = t == "gold" ? 30 : (t == "red" ? 20 : 10);

        _flagManager.SpawnFlag(new FlagSpawnPayload
        {
            flag_id = "F_" + System.DateTime.Now.Ticks, flag_type = t, score = sc,
            pos = new Vector3Json { x = pos.x, y = pos.y, z = pos.z }, lifetime = -1
        });
    }

    private void HandleServerMessage(string type, JObject payload) { /* online mode messages handled here */ }

    public void TriggerCountdown(int count) => OnCountdown?.Invoke(count);
    public void TriggerGameStarted() => OnGameStarted?.Invoke();
    public void TriggerScoreUpdated(ScoreUpdatePayload data) => OnScoreUpdated?.Invoke(data);
    public void TriggerGameEnded(GameEndPayload data) => OnGameEnded?.Invoke(data);
}