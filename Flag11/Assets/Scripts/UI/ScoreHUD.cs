using UnityEngine;
using UnityEngine.UI;

public class ScoreHUD : MonoBehaviour
{
    [Header("UI References")]
    public Text myScoreText;
    public Text opponentScoreText;
    public Text rankText;
    public GameObject doubleBuffIndicator;

    private GameManager _game;

    void Start()
    {
        _game = FindObjectOfType<GameManager>();
        if (_game != null)
        {
            _game.OnScoreUpdated += OnScoreUpdated;
        }
    }

    private void OnScoreUpdated(ScoreUpdatePayload data)
    {
            if (myScoreText != null)
                myScoreText.text = $"You: {_game.myScore}";

            if (opponentScoreText != null)
                opponentScoreText.text = $"Opponent: {_game.opponentScore}";

        if (doubleBuffIndicator != null)
            doubleBuffIndicator.SetActive(_game.myDoubleActive);

        if (rankText != null && data.scoreboard != null && data.scoreboard.Count > 0)
        {
            var ctx = GameContext.Instance;
            foreach (var entry in data.scoreboard)
            {
                if (entry.player_id == ctx.PlayerID)
                {
                    rankText.text = $"#{data.scoreboard.IndexOf(entry) + 1}";
                    break;
                }
            }
        }
    }

    void OnDestroy()
    {
        if (_game != null)
            _game.OnScoreUpdated -= OnScoreUpdated;
    }
}