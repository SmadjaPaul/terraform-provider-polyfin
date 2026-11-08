# Resource: polyfin_playback_languages

Owns only the default server-wide audio and subtitle language preferences.

```hcl
resource "polyfin_playback_languages" "family" {
  default_audio_language    = "fre"
  default_subtitle_language = "fre"
}
```

Polyfin v0.61.1's `PUT /admin/api/settings` is partial. This resource sends only
`defaultAudioLanguage` and `defaultSubtitleLanguage` and re-reads both
properties to verify. It never touches the persisted public address, provider
OIDC credentials, stored user preferences, playlists, or other server settings.

Values use ISO 639-2/**B**: `fre`, `eng`, `spa`, `jpn`, or `""`.
The server defaults only apply to users who did not choose a personal language.
Per-title remembered track choices can override those defaults.
It does **not** remove foreign tracks, hide subtitle entries, or promise
"always original language" for movies and anime. Version ordering remains
a distinct, currently unmanaged Polyfin setting.

Import existing settings with ID `server` in the approved isolated state.
Deletion intentionally fails to protect live configuration.
Upstream contract:
https://github.com/moodiness/polyfin/blob/v0.61.1/docs/playback.md#default-audio-and-subtitle-languages
