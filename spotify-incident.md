# Spotify pause/play error investigation

## Issue
Pause/Play actions show a generic error even though Spotify sometimes pauses/plays successfully.

## Observed behavior
- Frontend console logs show repeated `pause/resume request` followed by `pause/resume failed` and retry exhaustion.
- UI shows alert: "An error occurred. Please try again."
- Backend logs show Spotify responses:
  - 403 with body:
    {
      "error": {
        "status": 403,
        "message": "Player command failed: Restriction violated",
        "reason": "UNKNOWN"
      }
    }
  - 200 with a non-JSON body (random text). Current backend treats this as `SPOTIFY_UNAVAILABLE`.

## Why this is confusing
Spotify sometimes applies the command (play/pause changes), but still returns a 403 or a non-standard 200 response. The app treats any non-OK response as a failure and shows a generic error, even when the state updates to paused/playing.

## Not the cause
- Spotify Premium is active (so Premium requirement is not the blocker).

## Likely causes (still possible)
- Active device not controllable via Spotify Connect or not the active device when the request is sent.
- Playback restrictions on the track/region/context.
- Token scopes missing or not refreshed correctly after re-auth (`user-modify-playback-state`, `user-read-playback-state`).
- Spotify returns transient 403 while still executing the command (reported by other developers).

## Suggested fixes (implementation plan)
1) Backend: treat HTTP 200 with non-JSON body as success for player commands (Spotify normally returns 204).
2) Backend: map 403 "Restriction violated" to a specific error code (e.g. `SPOTIFY_RESTRICTION`).
3) Frontend: show a specific message for `SPOTIFY_RESTRICTION` and suppress the generic error if player state updates to the expected state shortly after.

## Suggested investigation (using the spike)
- Use the `spike-spotify-api` project to reproduce the error in isolation and capture the raw Spotify responses for play/pause.
- Compare the spike responses with the app backend responses to identify any proxy or formatting differences.

## Quick checks to run
1) Ensure a Spotify Connect device is active and selected before calling play/pause.
2) Confirm token scopes: `user-modify-playback-state`, `user-read-playback-state` after re-auth.
3) Try a different track/playlist to rule out content restriction.
4) Log `/me/player/devices` at the time of failure to confirm device state.

## References
- Spotify Web API: Pause Playback (requires Premium + scope `user-modify-playback-state`)
  https://developer.spotify.com/documentation/web-api/reference/pause-a-users-playback
- Community report: 403 "Restriction violated" even when pause/resume works
  https://community.spotify.com/t5/Spotify-for-Developers/start-resume-and-pause-player-throw-FORBIDDEN/td-p/6702509
