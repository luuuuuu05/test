using System.Collections.Generic;
using System.IO;
using UnityEngine;
using UnityEngine.AddressableAssets;
using UnityEngine.ResourceManagement.AsyncOperations;
using UnityEngine.UI;
using UnityEngine.XR;
using UnityEngine.XR.Interaction.Toolkit;

public class MapEditorManager : MonoBehaviour
{
    [Header("UI Panel")]
    public GameObject panel;
    public Transform itemListParent;
    public Button itemButtonPrefab;
    public Text statusText;

    [Header("Edit Panel")]
    public GameObject editPanel;
    public InputField posXInput, posYInput, posZInput;
    public InputField rotXInput, rotYInput, rotZInput;
    public Button deleteButton, duplicateButton;
    public Button saveButton, clearButton, backButton;

    [Header("XR Controllers")]
    public XRRayInteractor rightRayInteractor;

    [System.Serializable]
    public class PlaceableDef
    {
        public string id;
        public string displayName;
        public GameObject prefab;
        public AssetReferenceGameObject prefabRef;
        public Vector3 defaultScale = Vector3.one;
        public PrimitiveType fallbackPrimitive = PrimitiveType.Cube;
        public Color fallbackColor = Color.gray;
    }

    public List<PlaceableDef> placeables = new List<PlaceableDef>();

    [System.Serializable]
    public class PlacedObjectData
    {
        public string id;
        public float px, py, pz;
        public float rx, ry, rz;
        public float sx, sy, sz;
    }

    [System.Serializable]
    public class MapData
    {
        public string mapName;
        public List<PlacedObjectData> objects = new List<PlacedObjectData>();
    }

    private PlaceableDef _selectedDef;
    private GameObject _ghost;
    private Material _ghostMat;
    private GameObject _hoveredObj;
    private Material _hoveredOrigMat;
    private GameObject _selectedObj;
    private Material _selectedOrigMat;
    private List<GameObject> _placed = new List<GameObject>();
    private MapData _mapData = new MapData();
    private float _ghostRotY;
    private bool _selectPressed;

    void Start()
    {
        _mapData.mapName = "Map_" + System.DateTime.Now.ToString("yyyyMMdd_HHmmss");
        _ghostMat = new Material(Shader.Find("Standard"));
        _ghostMat.color = new Color(1, 1, 1, 0.3f);

        CreateItemButtons();
        if (saveButton != null) saveButton.onClick.AddListener(SaveMap);
        if (clearButton != null) clearButton.onClick.AddListener(ClearAll);
        if (backButton != null) backButton.onClick.AddListener(() =>
            UnityEngine.SceneManagement.SceneManager.LoadScene("MapSelect"));
        if (deleteButton != null) deleteButton.onClick.AddListener(DeleteSelected);
        if (duplicateButton != null) duplicateButton.onClick.AddListener(DuplicateSelected);
        if (editPanel != null) editPanel.SetActive(false);

        BindInputFields();

        // Use XRI event system instead of polling
        if (rightRayInteractor != null)
        {
            rightRayInteractor.hoverEntered.AddListener(OnHoverEnter);
            rightRayInteractor.hoverExited.AddListener(OnHoverExit);
            rightRayInteractor.selectEntered.AddListener(OnSelectEnter);
        }

        UpdateStatus("Select an item from the panel");
    }

    // ========== XRI Events ==========
    private void OnHoverEnter(HoverEnterEventArgs args)
    {
        var tag = args.interactableObject.transform.GetComponent<PlacedObjectTag>();
        if (tag == null || _selectedObj == tag.gameObject) return;
        HighlightObject(tag.gameObject, Color.yellow, 0.3f);
        _hoveredObj = tag.gameObject;
    }

    private void OnHoverExit(HoverExitEventArgs args)
    {
        var tag = args.interactableObject.transform.GetComponent<PlacedObjectTag>();
        if (tag == null) return;
        if (_hoveredObj == tag.gameObject)
        {
            RestoreHighlight(tag.gameObject, _hoveredOrigMat);
            _hoveredObj = null;
        }
    }

    private void OnSelectEnter(SelectEnterEventArgs args)
    {
        var tag = args.interactableObject.transform.GetComponent<PlacedObjectTag>();
        if (tag != null)
        {
            // Selected a placed object
            if (_hoveredObj == tag.gameObject)
            {
                RestoreHighlight(tag.gameObject, _hoveredOrigMat);
                _hoveredObj = null;
            }
            SelectObject(tag.gameObject);
        }
    }

    // ========== UI ==========
    private void CreateItemButtons()
    {
        if (itemListParent == null || itemButtonPrefab == null) return;
        foreach (var def in placeables)
        {
            var btn = Instantiate(itemButtonPrefab, itemListParent);
            btn.gameObject.SetActive(true);
            var txt = btn.GetComponentInChildren<Text>();
            if (txt != null) txt.text = def.displayName;
            var d = def;
            btn.onClick.AddListener(() => SelectPlaceable(d));
        }
    }

    private void SelectPlaceable(PlaceableDef def)
    {
        DeselectObject();
        _selectedDef = def;
        CreateGhost(def);
        UpdateStatus("Placing: " + def.displayName);
    }

    // ========== Ghost ==========
    private void CreateGhost(PlaceableDef def)
    {
        if (_ghost != null) Destroy(_ghost);

        if (def.prefab != null)
        {
            _ghost = Instantiate(def.prefab);
            foreach (var c in _ghost.GetComponentsInChildren<Collider>())
                Destroy(c);
        }
        else
        {
            _ghost = GameObject.CreatePrimitive(def.fallbackPrimitive);
            if (_ghost.TryGetComponent<Collider>(out var c))
                Destroy(c);
        }

        _ghost.transform.localScale = def.defaultScale;
        foreach (var r in _ghost.GetComponentsInChildren<Renderer>())
        {
            r.sharedMaterial = _ghostMat;
            r.sharedMaterial.color = new Color(def.fallbackColor.r, def.fallbackColor.g, def.fallbackColor.b, 0.3f);
        }
        _ghost.name = "Ghost_" + def.id;
    }

    // ========== Update (only ghost position poll) ==========
    void Update()
    {
        UpdateGhostRotation();
        UpdateGhostPosition();
        HandlePlacementOnEmpty();
    }

    private void UpdateGhostRotation()
    {
        if (_ghost == null || _selectedDef == null) return;
        var device = InputDevices.GetDeviceAtXRNode(XRNode.RightHand);
        if (device.TryGetFeatureValue(CommonUsages.primary2DAxis, out Vector2 stick))
        {
            if (Mathf.Abs(stick.x) > 0.2f)
                _ghostRotY += stick.x * 120f * Time.deltaTime;
        }
    }

    private void UpdateGhostPosition()
    {
        if (_ghost == null || _selectedDef == null) return;

        if (rightRayInteractor != null && rightRayInteractor.TryGetCurrent3DRaycastHit(out var hit))
        {
            _ghost.transform.position = hit.point;
        }
        else
        {
            var cam = Camera.main;
            if (cam != null)
                _ghost.transform.position = cam.transform.position + cam.transform.forward * 3f;
        }
        _ghost.transform.rotation = Quaternion.Euler(0, _ghostRotY, 0);

        var r = _ghost.GetComponentInChildren<Renderer>();
        if (r != null)
        {
            float bottomOffset = _ghost.transform.position.y - r.bounds.min.y;
            _ghost.transform.position += Vector3.up * bottomOffset;
        }
    }

    private void HandlePlacementOnEmpty()
    {
        if (_selectedDef == null || _ghost == null) return;

        var ctrl = rightRayInteractor != null ?
            rightRayInteractor.GetComponent<ActionBasedController>() : null;
        if (ctrl == null) return;

        bool pressed = false;
        if (ctrl.activateAction.action != null)
            pressed = ctrl.activateAction.action.WasPressedThisFrame();
        if (!pressed && ctrl.selectAction.action != null)
            pressed = ctrl.selectAction.action.WasPressedThisFrame();

        if (!pressed) return;

        // Only place if not hovering an interactable
        if (rightRayInteractor.hasHover) return;

        PlaceCurrent();
    }

    // ========== Place ==========
    public void PlaceCurrent()
    {
        if (_selectedDef == null || _ghost == null) return;

        Vector3 pos = _ghost.transform.position;
        Vector3 rot = _ghost.transform.eulerAngles;
        Vector3 scl = _selectedDef.defaultScale;

        GameObject go;
        if (_selectedDef.prefab != null)
        {
            go = Instantiate(_selectedDef.prefab);
        }
        else
        {
            go = GameObject.CreatePrimitive(_selectedDef.fallbackPrimitive);
            go.GetComponent<Renderer>().sharedMaterial = new Material(Shader.Find("Standard"));
            go.GetComponent<Renderer>().sharedMaterial.color = _selectedDef.fallbackColor;
        }
        go.name = _selectedDef.id + "_" + _placed.Count;
        go.transform.position = pos;
        go.transform.eulerAngles = rot;
        go.transform.localScale = scl;
        FinalizePlacedObject(go, _selectedDef.id);
    }

    private void FinalizePlacedObject(GameObject go, string id)
    {
        go.AddComponent<PlacedObjectTag>().placeableId = id;

        if (go.GetComponent<Collider>() == null)
            go.AddComponent<BoxCollider>();

        if (go.GetComponent<Rigidbody>() == null)
        {
            var rb = go.AddComponent<Rigidbody>();
            rb.isKinematic = true;
        }

        var grab = go.AddComponent<XRGrabInteractable>();

        _placed.Add(go);
        _mapData.objects.Add(new PlacedObjectData
        {
            id = id,
            px = go.transform.position.x, py = go.transform.position.y, pz = go.transform.position.z,
            rx = go.transform.eulerAngles.x, ry = go.transform.eulerAngles.y, rz = go.transform.eulerAngles.z,
            sx = go.transform.localScale.x, sy = go.transform.localScale.y, sz = go.transform.localScale.z
        });

        UpdateStatus("Placed: " + id + " (" + _placed.Count + " total)");
    }

    // ========== Highlight / Select / Deselect ==========
    private void HighlightObject(GameObject obj, Color color, float emission)
    {
        var r = obj.GetComponentInChildren<Renderer>();
        if (r == null) return;
        _hoveredOrigMat = r.sharedMaterial;
        var mat = new Material(Shader.Find("Standard"));
        mat.color = color;
        mat.EnableKeyword("_EMISSION");
        mat.SetColor("_EmissionColor", color * emission);
        r.sharedMaterial = mat;
    }

    private void RestoreHighlight(GameObject obj, Material origMat)
    {
        if (obj == null || origMat == null) return;
        var r = obj.GetComponentInChildren<Renderer>();
        if (r != null) r.sharedMaterial = origMat;
    }

    private void SelectObject(GameObject obj)
    {
        if (_selectedObj == obj) return;
        DeselectObject();

        _selectedObj = obj;
        HighlightObject(obj, Color.cyan, 0.5f);
        _selectedOrigMat = _hoveredOrigMat;

        if (editPanel != null) editPanel.SetActive(true);
        PopulateInputFields();
        UpdateStatus("Selected: " + obj.name);
    }

    private void DeselectObject()
    {
        if (_selectedObj != null)
        {
            RestoreHighlight(_selectedObj, _selectedOrigMat);
            _selectedObj = null;
        }
        if (editPanel != null) editPanel.SetActive(false);
    }

    // ========== Delete / Duplicate ==========
    private void DeleteSelected()
    {
        if (_selectedObj == null) return;
        string name = _selectedObj.name;
        _placed.Remove(_selectedObj);
        Destroy(_selectedObj);
        _selectedObj = null;
        if (editPanel != null) editPanel.SetActive(false);
        RebuildMapData();
        UpdateStatus("Deleted: " + name);
    }

    private void DuplicateSelected()
    {
        if (_selectedObj == null) return;
        var clone = Instantiate(_selectedObj, _selectedObj.transform.parent);
        clone.transform.position += new Vector3(0.3f, 0, 0);
        clone.name = _selectedObj.name + "_copy";
        _placed.Add(clone);
        SelectObject(clone);
        RebuildMapData();
        UpdateStatus("Duplicated");
    }

    // ========== Position/Rotation Input ==========
    private void BindInputFields()
    {
        if (posXInput != null) posXInput.onValueChanged.AddListener(_ => ApplyInputFields());
        if (posYInput != null) posYInput.onValueChanged.AddListener(_ => ApplyInputFields());
        if (posZInput != null) posZInput.onValueChanged.AddListener(_ => ApplyInputFields());
        if (rotXInput != null) rotXInput.onValueChanged.AddListener(_ => ApplyInputFields());
        if (rotYInput != null) rotYInput.onValueChanged.AddListener(_ => ApplyInputFields());
        if (rotZInput != null) rotZInput.onValueChanged.AddListener(_ => ApplyInputFields());
    }

    private void PopulateInputFields()
    {
        if (_selectedObj == null) return;
        var p = _selectedObj.transform.position;
        var r = _selectedObj.transform.eulerAngles;
        if (posXInput != null) posXInput.text = p.x.ToString("F2");
        if (posYInput != null) posYInput.text = p.y.ToString("F2");
        if (posZInput != null) posZInput.text = p.z.ToString("F2");
        if (rotXInput != null) rotXInput.text = r.x.ToString("F1");
        if (rotYInput != null) rotYInput.text = r.y.ToString("F1");
        if (rotZInput != null) rotZInput.text = r.z.ToString("F1");
    }

    private void ApplyInputFields()
    {
        if (_selectedObj == null) return;
        float px = ParseInput(posXInput, _selectedObj.transform.position.x);
        float py = ParseInput(posYInput, _selectedObj.transform.position.y);
        float pz = ParseInput(posZInput, _selectedObj.transform.position.z);
        float rx = ParseInput(rotXInput, _selectedObj.transform.eulerAngles.x);
        float ry = ParseInput(rotYInput, _selectedObj.transform.eulerAngles.y);
        float rz = ParseInput(rotZInput, _selectedObj.transform.eulerAngles.z);
        _selectedObj.transform.position = new Vector3(px, py, pz);
        _selectedObj.transform.eulerAngles = new Vector3(rx, ry, rz);
        RebuildMapData();
    }

    private float ParseInput(InputField field, float fallback)
    {
        if (field == null) return fallback;
        return float.TryParse(field.text, out var v) ? v : fallback;
    }

    // ========== Save / Clear ==========
    private void SaveMap()
    {
        RebuildMapData();
        Debug.Log("[MapEditor] SaveMap: " + _mapData.objects.Count + " objects, name=" + _mapData.mapName);
        var sceneObjects = new List<SceneObject>();
        foreach (var obj in _mapData.objects) {
            sceneObjects.Add(new SceneObject {
                id = obj.id,
                prefab = obj.id,
                pos = new Vector3Json { x=obj.px, y=obj.py, z=obj.pz },
                rot = new Vector3Json { x=obj.rx, y=obj.ry, z=obj.rz },
                scale = new Vector3Json { x=obj.sx, y=obj.sy, z=obj.sz }
            });
        }
        LocalMapStore.Save(_mapData.mapName, sceneObjects);
        UpdateStatus("Saved: " + _mapData.mapName + " (" + _mapData.objects.Count + " objects)");
    }

    private void ClearAll()
    {
        foreach (var go in _placed) Destroy(go);
        _placed.Clear();
        _mapData.objects.Clear();
        DeselectObject();
        UpdateStatus("Cleared");
    }

    private void RebuildMapData()
    {
        _mapData.objects.Clear();
        foreach (var go in _placed)
        {
            if (go == null) continue;
            var tag = go.GetComponent<PlacedObjectTag>();
            _mapData.objects.Add(new PlacedObjectData
            {
                id = tag != null ? tag.placeableId : "unknown",
                px = go.transform.position.x, py = go.transform.position.y, pz = go.transform.position.z,
                rx = go.transform.eulerAngles.x, ry = go.transform.eulerAngles.y, rz = go.transform.eulerAngles.z,
                sx = go.transform.localScale.x, sy = go.transform.localScale.y, sz = go.transform.localScale.z
            });
        }
    }

    private void UpdateStatus(string msg)
    {
        if (statusText != null) statusText.text = msg;
        Debug.Log("[MapEditor] " + msg);
    }
}

public class PlacedObjectTag : MonoBehaviour
{
    public string placeableId;
}
