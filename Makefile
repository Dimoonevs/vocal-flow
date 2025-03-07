HOST=13.61.187.160
HOMEDIR=/var/www/video-ai/
USER=dima

video-ai-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/video-ai-linux-amd64 ./

upload-video-ai: video-ai-linux
	rsync -rzv --progress --rsync-path="sudo rsync" \
		./bin/video-ai-linux-amd64  \
		./utils/cfg/prod.ini \
		./utils/restart.sh \
		$(USER)@$(HOST):$(HOMEDIR)

restart-video-ai:
	echo "sudo su && cd $(HOMEDIR) && bash restart.sh && exit" | ssh $(USER)@$(HOST) /bin/sh

upload-and-restart: upload-video-ai restart-video-ai

run-local:
	go run main.go -config ./utils/cfg/local.ini