using UnityEngine;
using UnityEngine.SceneManagement;

public class SceneLoader : MonoBehaviour
{
    public void LoadScene(string sceneName)
    {
        SceneManager.LoadScene(sceneName);
    }

    public void DevTestLoadGame()
    {
        var ctx = GameContext.Instance;
        if (ctx != null)
        {
            ctx.PlayerID = "DEV_" + System.Guid.NewGuid().ToString("N").Substring(0, 4);
            ctx.displayName = "DevPlayer";
            ctx.serverIP = "";
            ctx.SetRoom("DEV_ROOM", ctx.PlayerID);
        }
        SceneManager.LoadScene("MRGame");
    }

    public void SoloDebugStart()
    {
        var ctx = GameContext.Instance ?? FindObjectOfType<GameContext>();
        if (ctx == null)
        {
            var go = new GameObject("GameContext_Solo");
            ctx = go.AddComponent<GameContext>();
            DontDestroyOnLoad(go);
        }

        ctx.PlayerID = "SOLO_" + Random.Range(1000, 9999);
        ctx.displayName = "Solo";
        ctx.serverIP = "";
        ctx.SetRoom("SOLO", ctx.PlayerID);

        PlayerPrefs.SetInt("SoloMode", 0);
        PlayerPrefs.SetInt("BuildMode", 0);
        SceneManager.LoadScene("MapSelect");
    }
}