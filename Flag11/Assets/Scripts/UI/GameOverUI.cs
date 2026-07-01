using UnityEngine;
using UnityEngine.UI;
using UnityEngine.SceneManagement;

public class GameOverUI : MonoBehaviour
{
    [Header("UI References")]
    public GameObject gameOverPanel;
    public Text resultTitleText;
    public Text myScoreText;
    public Text opponentScoreText;
    public Text rankText;
    public Button returnButton;

    private GameManager _game;

    void Start()
    {
        if (gameOverPanel != null) gameOverPanel.SetActive(false);

        _game = FindObjectOfType<GameManager>();
        if (_game != null)
        {
            _game.OnGameEnded += OnGameEnded;
        }

        if (returnButton != null)
            returnButton.onClick.AddListener(() => SceneManager.LoadScene("MainMenu"));
    }

    private void OnGameEnded(GameEndPayload data)
    {
        Debug.Log("[GameOver] Showing results");

        if (gameOverPanel != null) gameOverPanel.SetActive(true);

        var ctx = GameContext.Instance;
        bool iWon = false;

        if (data.final_scores != null && data.final_scores.Count > 0 && ctx != null)
        {
            iWon = data.final_scores[0].player_id == ctx.PlayerID;
        }
        else
        {
            iWon = _game.myScore > _game.opponentScore;
        }

        if (resultTitleText != null)
        {
            resultTitleText.text = iWon ? "VICTORY!" : "DEFEAT";
            resultTitleText.color = iWon ? new Color(1f, 0.84f, 0f) : Color.red;
        }

        if (myScoreText != null)
            myScoreText.text = $"Your Score: {_game.myScore}";

        if (opponentScoreText != null)
            opponentScoreText.text = $"Opponent: {_game.opponentScore}";

        if (rankText != null && ctx != null)
        {
            int rank = 0;
            if (data.final_scores != null)
            {
                foreach (var entry in data.final_scores)
                {
                    if (entry.player_id == ctx.PlayerID)
                    {
                        rank = entry.rank;
                        break;
                    }
                }
            }
            rankText.text = rank > 0 ? $"Rank #{rank}" : "";
        }
    }

    void OnDestroy()
    {
        if (_game != null)
            _game.OnGameEnded -= OnGameEnded;
    }
}