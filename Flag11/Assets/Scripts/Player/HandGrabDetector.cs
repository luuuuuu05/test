using UnityEngine;
using UnityEngine.XR.Interaction.Toolkit;

public class HandGrabDetector : MonoBehaviour
{
    private XRDirectInteractor _interactor;
    private GameManager _game;
    private bool _isGrabbing;

    void Start()
    {
        _interactor = GetComponent<XRDirectInteractor>();
        if (_interactor == null)
            _interactor = gameObject.AddComponent<XRDirectInteractor>();

        _game = FindObjectOfType<GameManager>();

        if (_interactor != null)
        {
            _interactor.selectEntered.AddListener(OnGrab);
            _interactor.selectExited.AddListener(OnRelease);
        }
    }

    private void OnGrab(SelectEnterEventArgs args)
    {
        var flag = args.interactableObject?.transform?.GetComponent<FlagObject>();
        if (flag != null && _game != null)
        {
            _isGrabbing = true;
            _game.OnFlagGrabbed(flag);
            Debug.Log($"[HandGrab] Grabbed flag: {flag.FlagID}");
        }
    }

    private void OnRelease(SelectExitEventArgs args)
    {
        _isGrabbing = false;
    }

    public bool IsGrabbing => _isGrabbing;

    void OnDestroy()
    {
        if (_interactor != null)
        {
            _interactor.selectEntered.RemoveListener(OnGrab);
            _interactor.selectExited.RemoveListener(OnRelease);
        }
    }
}