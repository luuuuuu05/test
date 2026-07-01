using System.Collections;
using UnityEngine;
using UnityEngine.UI;

public class CountdownUI : MonoBehaviour
{
    public Text countdownText;
    public GameObject countdownPanel;

    private GameManager _game;

    void Start()
    {
        _game = FindObjectOfType<GameManager>();
        if (_game != null)
        {
            _game.OnCountdown += OnCountdown;
            _game.OnGameStarted += () =>
            {
                if (countdownPanel != null) countdownPanel.SetActive(false);
            };
        }

        if (countdownPanel != null) countdownPanel.SetActive(false);
    }

    private void OnCountdown(int count)
    {
        if (countdownPanel != null) countdownPanel.SetActive(true);

        if (countdownText != null)
        {
            countdownText.text = count > 0 ? count.ToString() : "GO!";
            countdownText.fontSize = count > 0 ? 200 : 150;
        }

        if (count == 0)
        {
            StartCoroutine(HideCountdownDelayed());
        }
    }

    private IEnumerator HideCountdownDelayed()
    {
        yield return new WaitForSeconds(1f);
        if (countdownPanel != null) countdownPanel.SetActive(false);
    }

    void OnDestroy()
    {
        if (_game != null)
            _game.OnCountdown -= OnCountdown;
    }
}