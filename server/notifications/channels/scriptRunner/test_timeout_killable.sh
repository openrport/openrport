#!/usr/bin/env sh

x=1
while [ $x -le 20 ]
do
  # echo "Welcome $x times"
  x=$(( $x + 1 ))
  sleep 1
done