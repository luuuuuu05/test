using System.Collections;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.SceneManagement;
using UnityEngine.UI;

public enum MRGameState { Idle, Countdown, Playing, Ended }

public class MRGameManager : MonoBehaviour
{
    public MRGameState State { get; private set; } = MRGameState.Idle;

    [Header("Game Settings")]
    public float gameDuration = 30f;
    public float respawnDelay = 3f;

    [Header("Flag Prefabs")]
    public GameObject goldFlagPrefab;
    public GameObject redFlagPrefab;
    public GameObject whiteFlagPrefab;

    [Header("Map Objects")]
    public SceneObjectRegistry objectRegistry;

    [Header("Score")]
    public int score;

    [Header("UI")]
    public Text scoreText, timerText, countdownText, finalScoreText;
    public GameObject countdownPanel, gameOverPanel;

    private float _gameStartTime;
    private int _flagCount;

    void Start()
    {
        WireBackButton();
        if (countdownPanel != null) countdownPanel.SetActive(false);
        if (gameOverPanel != null) gameOverPanel.SetActive(false);
        LoadSelectedMap();
        StartGame();
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

    private void LoadSelectedMap()
    {
        MapLoaderHelper.LoadMapIntoScene(objectRegistry);
    }

    public void StartGame()
    {
        State = MRGameState.Countdown;
        score = 0;
        if (scoreText != null) scoreText.text = "Score: 0";
        SpawnFlags();
        StartCoroutine(GameFlow());
    }

    private IEnumerator GameFlow()
    {
        if (countdownPanel != null) countdownPanel.SetActive(true);
        for (int i = 3; i > 0; i--)
        {
            if (countdownText != null) countdownText.text = i.ToString();
            yield return new WaitForSeconds(1f);
        }
        if (countdownText != null) countdownText.text = "GO!";
        yield return new WaitForSeconds(1f);
        if (countdownPanel != null) countdownPanel.SetActive(false);

        State = MRGameState.Playing;
        _gameStartTime = Time.time;
        while (Time.time - _gameStartTime < gameDuration)
        {
            if (timerText != null)
                timerText.text = (int)(gameDuration - (Time.time - _gameStartTime)) + "s";
            yield return null;
        }

        State = MRGameState.Ended;
        if (timerText != null) timerText.text = "0s";
        if (gameOverPanel != null) gameOverPanel.SetActive(true);
    }

    public void OnFlagGrabbed(MRFlag flag)
    {
        if (State != MRGameState.Playing) return;
        score += flag.score;
        if (scoreText != null) scoreText.text = "Score: " + score;
        flag.gameObject.SetActive(false);
        StartCoroutine(RespawnFlag(flag));
    }

    private IEnumerator RespawnFlag(MRFlag flag)
    {
        yield return new WaitForSeconds(respawnDelay);
        if (State != MRGameState.Playing) yield break;
        flag.transform.position = new Vector3(Random.Range(-3f, 3f), 0.3f, Random.Range(0.5f, 3f));
        flag.ResetGrab();
        flag.gameObject.SetActive(true);
    }

    private void SpawnFlags()
    {
        string[] types = { "gold", "red", "white", "white", "white" };
        int[] scores = { 30, 20, 10, 10, 10 };
        Color[] colors = { new Color(1f,0.84f,0f), Color.red, Color.white, Color.white, Color.white };
        float[] xs = { -2f, -1f, 0f, 1f, 2f };
        for (int i = 0; i < types.Length; i++)
        {
            var go = new GameObject();
            go.transform.position = new Vector3(xs[i], 0.3f, 1.5f);
            var flag = go.AddComponent<MRFlag>();
            GameObject vp = types[i] switch { "gold" => goldFlagPrefab, "red" => redFlagPrefab, _ => whiteFlagPrefab };
            flag.Setup(types[i], scores[i], colors[i], vp);
        }
    }
}
