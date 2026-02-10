# Spotify Premium Requirement (Post-Login Guard)

## Goal
Ensure the app rejects non-Premium Spotify accounts immediately after OAuth login, and logs the user out. The product has no value for non-Premium users because playback control is blocked by Spotify.

## Current Behavior
- Non-Premium users can complete OAuth and appear as authenticated.
- Playback commands later fail with `SPOTIFY_FORBIDDEN`.
- This creates confusion during session join and sync.

## Desired Behavior
After Spotify OAuth completes:
1. Detect if the authenticated account is Premium.
2. If not Premium, immediately log the user out server-side.
3. Redirect to the frontend with an explicit error code.
4. UI should show a clear message and prevent session creation/join.

## Implementation Notes
- Use Spotify `GET /v1/me` response to read the `product` field.
- Treat `product != "premium"` as non-Premium.
- Logout path should clear session cookies and spotify_user_id.
- Redirect example: `/?auth_error=spotify_premium_required`.

## Proposed Backend Touchpoints
- `backend/internal/spotify/token_service.go`: extend `UserInfoResponse` to capture `product`.
- `backend/internal/handlers/auth.go`: in callback, if product is not premium, clear session and redirect with error.

## Proposed Frontend Touchpoints
- Login UI should surface the error code from query params and show a dedicated message.
- Disable session create/join when the error is present.

## Acceptance Criteria
- Non-Premium user finishes OAuth and is immediately logged out.
- UI shows a specific "Spotify Premium required" message.
- No session creation/join possible for non-Premium users.
