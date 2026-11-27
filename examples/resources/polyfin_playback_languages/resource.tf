resource "polyfin_playback_languages" "family" {
  # ISO 639-2/B codes: fre, eng, spa, jpn, or "" for no preference.
  default_audio_language    = "fre"
  default_subtitle_language = "fre"
}

# The server-wide defaults are a singleton: the provider refuses to destroy
# them, so removing the block from the configuration keeps the Polyfin value.
