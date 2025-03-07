#!/usr/bin/env bash

sudo pm2 stop video-ai
sudo GOMAXPROCS=3 pm2 start video-ai-linux-amd64 --name=video-ai -- -config=./prod.ini
sudo pm2 save