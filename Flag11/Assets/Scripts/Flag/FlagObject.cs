using UnityEngine;
using UnityEngine.XR.Interaction.Toolkit;

public class FlagObject : MonoBehaviour
{
    public string FlagID { get; private set; }
    public string FlagType { get; private set; }
    public int Score { get; private set; }

    public GameObject visual;
    public GameObject goldPrefab;
    public GameObject redPrefab;
    public GameObject whitePrefab;
    public GameObject doublePrefab;

    private XRGrabInteractable _grab;
    private bool _grabbed;
    private MultiGameManager _multiGM;
    private GameManager _gm;

    public void Initialize(FlagSpawnPayload data)
    {
        FlagID = data.flag_id;
        FlagType = data.flag_type;
        Score = data.score;
        name = "Flag_" + data.flag_id;
        transform.position = new Vector3(data.pos.x, data.pos.y, data.pos.z);

        _multiGM = FindObjectOfType<MultiGameManager>();
        if (_multiGM == null)
            _gm = FindObjectOfType<GameManager>();

        SetupVisual();
        SetupCollision();
    }

    private void SetupVisual()
    {
        GameObject prefab = FlagType switch { "gold" => goldPrefab, "red" => redPrefab, "double" => doublePrefab, _ => whitePrefab };

        if (prefab != null)
        {
            visual = Instantiate(prefab, transform);
            visual.transform.localScale = new Vector3(0.3f, 0.8f, 0.3f);
            foreach (var c in visual.GetComponentsInChildren<Collider>())
                Destroy(c);
        }
        else
        {
            visual = GameObject.CreatePrimitive(PrimitiveType.Cylinder);
            visual.transform.SetParent(transform);
            visual.transform.localPosition = Vector3.zero;
            visual.transform.localScale = new Vector3(0.3f, 0.8f, 0.3f);
            var r = visual.GetComponent<Renderer>();
            if (r != null)
            {
                r.material = new Material(Shader.Find("Standard"));
                r.material.color = FlagType switch { "gold" => new Color(1f, 0.84f, 0f), "red" => Color.red, "double" => Color.magenta, _ => Color.white };
            }
        }
    }

    private void SetupCollision()
    {
        var col = gameObject.AddComponent<BoxCollider>();
        col.size = new Vector3(0.8f, 1.0f, 0.8f);

        var rb = gameObject.AddComponent<Rigidbody>();
        rb.isKinematic = true;
        rb.useGravity = false;

        _grab = gameObject.AddComponent<XRGrabInteractable>();
        _grab.selectEntered.AddListener(OnGrabbed);
    }

    private void OnGrabbed(SelectEnterEventArgs args)
    {
        if (_grabbed) return;
        _grabbed = true;

        if (_multiGM != null)
        {
            _multiGM.OnFlagGrabbed(this);
            return;
        }

        if (_gm != null) _gm.OnFlagGrabbed(this);
    }

    private void OnTriggerEnter(Collider other)
    {
        if (_grabbed) return;
        if (other.isTrigger) return;

        if (_multiGM != null)
        {
            if (_multiGM.State != MultiState.Playing) return;
            _grabbed = true;
            Debug.Log($"[Flag] Proximity grab: {FlagID} ({FlagType}, {Score}pts) by {other.name}");
            _multiGM.OnFlagGrabbed(this);
            return;
        }

        if (_gm == null) return;
        if (_gm.State.CurrentState != GameState.Playing) return;

        _grabbed = true;
        Debug.Log($"[Flag] Proximity grab: {FlagID} ({FlagType}, {Score}pts) by {other.name}");
        _gm.OnFlagGrabbed(this);
    }

    void OnDestroy() { if (visual != null) Destroy(visual); }
}