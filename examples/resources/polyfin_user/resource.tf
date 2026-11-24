resource "random_password" "svc_music_assistant" {
  length  = 32
  special = false
}

resource "polyfin_user" "svc_music_assistant" {
  name             = "svc-music-assistant"
  password_wo      = random_password.svc_music_assistant.result
  is_administrator = false
  is_hidden        = true

  # Music Assistant treats tracks as unavailable without the download permission.
  enable_content_downloading = true
}
