# Pilot

Pilot is a web app for playing media from a local hard drive. It scans a local folder and presents
the files for download, playback in browser, or on a connected TV. It uses CEC to let you pause,
fast-forward/rewind, and control playback with the TV remote.

TV playback assumes a VLC server is running on port 8081, for example:
`> DISPLAY=:0 cvlc -I http --http-host "127.0.0.1" --http-port 8081 --http-password="raspberry"`
`> pilot -addr :8080 -root /mnt/media -folders TV,Movies`