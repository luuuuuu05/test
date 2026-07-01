using System.Collections;
using UnityEngine;
using UnityEngine.XR.Interaction.Toolkit;

public class MRFlag : MonoBehaviour
{
    public string flagType;
    public int score;
    public Color flagColor;

    private XRGrabInteractable _grab;
    private bool _grabbed;

    public void Setup(string type, int pts, Color color, GameObject visualPrefab = null)
    {
        flagType = type;
        score = pts;
        flagColor = color;
        name = "Flag_" + type;

        if (visualPrefab != null)
        {
            var body = Instantiate(visualPrefab, transform);
            body.transform.localPosition = Vector3.zero;
            foreach (var c in body.GetComponentsInChildren<Collider>())
                Destroy(c);
        }
        else
        {
            var body = GameObject.CreatePrimitive(PrimitiveType.Cylinder);
            body.transform.SetParent(transform);
            body.transform.localPosition = Vector3.zero;
            body.transform.localScale = new Vector3(0.2f, 0.6f, 0.2f);
            body.GetComponent<Renderer>().material.color = flagColor;
        }

        var col = gameObject.AddComponent<BoxCollider>();
        col.size = new Vector3(0.8f, 1.0f, 0.8f);

        var rb = gameObject.AddComponent<Rigidbody>();
        rb.isKinematic = true;

        _grab = gameObject.AddComponent<XRGrabInteractable>();
        _grab.selectEntered.AddListener(OnGrabbed);
    }

    public void ResetGrab()
    {
        _grabbed = false;
    }

    private void OnGrabbed(SelectEnterEventArgs args)
    {
        if (_grabbed) return;
        _grabbed = true;
        var gm = FindObjectOfType<MRGameManager>();
        if (gm != null) gm.OnFlagGrabbed(this);
    }
}
