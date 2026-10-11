#!/usr/bin/env bash
tmux new-session -d -s demo 'go run .'
tmux split-window -h 'sleep 1; redis-cli -p 8080'
tmux split-window -v 'sleep 1; redis-cli -p 8080'
tmux attach -t demo
