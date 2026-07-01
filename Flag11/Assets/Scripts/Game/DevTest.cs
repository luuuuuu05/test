using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class DevTest : MonoBehaviour
{
    private GameManager _game;
    private FlagManager _flagManager;

    void Start()
    {
        _game = GetComponent<GameManager>();
        _flagManager = GetComponent<FlagManager>();

        var ctx = GameContext.Instance;
        if (ctx != null && string.IsNullOrEmpty(ctx.serverIP))
        {
            var mapSelector = GetComponent<MapSelector>();
            if (mapSelector != null)
            {
                Debug.Log("[DevTest] Solo mode - showing MapSelector");
                mapSelector.Show();
            }
            else
            {
                Debug.Log("[DevTest] No MapSelector, starting default game");
                StartGame("official_default");
            }
        }
        else
        {
            Debug.Log("[DevTest] Online mode - waiting for server events");
        }
    }

    public void StartGame(string mapName)
    {
        Debug.Log($"[DevTest] Starting game with map: {mapName}");
        
        var ctx = GameContext.Instance;
        if (ctx != null && !string.IsNullOrEmpty(ctx.serverIP))
        {
            Debug.Log("[DevTest] Online mode - skipping local game start");
            return;
        }

        StartCoroutine(RunGame());
    }

    public void StartBuildMode()
    {
        Debug.Log("[DevTest] Build mode - loading MapEditor scene");
        UnityEngine.SceneManagement.SceneManager.LoadScene("MapEditor");
    }

    private IEnumerator RunGame()
    {
        _flagManager.ClearAll();

        string[] types = { "gold", "red", "white", "white", "white" };
        float[] xs = { -2f, -1f, 0f, 1f, 2f };
        for (int i = 0; i < types.Length; i++)
        {
            _flagManager.SpawnFlag(new FlagSpawnPayload
            {
                flag_id = "F_" + i, flag_type = types[i],
                score = types[i] == "gold" ? 30 : (types[i] == "red" ? 20 : 10),
                pos = new Vector3Json { x = xs[i], y = 0.3f, z = 1.5f },
                flag_point_id = "FP_" + i, lifetime = -1
            });
        }

        _game.State.ForceState(GameState.Waiting);
        yield return new WaitForSeconds(1f);

        _game.State.ForceState(GameState.Countdown);
        for (int i = 3; i >= 0; i--) { _game.TriggerCountdown(i); yield return new WaitForSeconds(1f); }

        _game.State.ForceState(GameState.Playing);
        _game.TriggerGameStarted();
        Debug.Log("[DevTest] Game started. Grab flags with controller trigger!");

        yield return new WaitForSeconds(30f);

        _game.State.ForceState(GameState.Ended);
        _game.TriggerGameEnded(new GameEndPayload
        {
            reason = "timeout", duration_actual = 30,
            final_scores = new List<GameResultEntry> { new GameResultEntry { rank = 1, player_id = GameContext.Instance?.PlayerID ?? "P1", display_name = "You", score = _game.myScore, flags_grabbed = 3, award = "champion" } }
        });
        Debug.Log("[DevTest] Game over");
    }
}
