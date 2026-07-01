using System;

public enum GameState
{
    Idle,
    Waiting,
    Countdown,
    Playing,
    Ended
}

public class GameStateMachine
{
    public GameState CurrentState { get; private set; } = GameState.Idle;

    public event Action<GameState, GameState> OnStateChanged;

    public bool CanTransitionTo(GameState next)
    {
        return CurrentState switch
        {
            GameState.Idle => next == GameState.Waiting,
            GameState.Waiting => next == GameState.Countdown || next == GameState.Idle,
            GameState.Countdown => next == GameState.Playing || next == GameState.Idle,
            GameState.Playing => next == GameState.Ended || next == GameState.Idle,
            GameState.Ended => next == GameState.Waiting || next == GameState.Idle,
            _ => false
        };
    }

    public void TransitionTo(GameState next)
    {
        if (!CanTransitionTo(next))
        {
            UnityEngine.Debug.LogWarning($"[State] Invalid transition: {CurrentState} -> {next}");
            return;
        }

        var prev = CurrentState;
        CurrentState = next;
        UnityEngine.Debug.Log($"[State] {prev} -> {next}");
        OnStateChanged?.Invoke(prev, next);
    }

    public void ForceState(GameState state)
    {
        CurrentState = state;
    }
}