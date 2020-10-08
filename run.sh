#!/bin/bash

# turn on bash's job control
#set -m

# Start the primary process and put it in the background
nohup ./message-server run >>/tmp/message.log 2>&1 &

# Start the helper process
nohup ./logic-server run >>/tmp/logic.log 2>&1 &

# the my_helper_process might need to know how to wait on the
# primary process to start before it does its work and returns


# now we bring the primary process back into the foreground
# and leave it there
#fg %1

while sleep 30; do
  ps aux |grep message-server |grep -v grep >/dev/null 2>&1
  PROCESS_1_STATUS=$?
  ps aux |grep logic-server |grep -v grep >/dev/null 2>&1
  PROCESS_2_STATUS=$?
  # If the greps above find anything, they exit with 0 status
  # If they are not both 0, then something is wrong
  if [ $PROCESS_1_STATUS -ne 0 -o $PROCESS_2_STATUS -ne 0 ]; then
    tail -n 50 /tmp/message.log
    tail -n 50 /tmp/logic.log
    echo "One of the processes has already exited."
    exit 1
  fi
  tail -n 50 /tmp/message.log
  tail -n 50 /tmp/logic.log
done