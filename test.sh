go build -o pilot *.go &&
./pilot \
-addr :8080 \
-root /mnt/media \
-folders TV,Movies
