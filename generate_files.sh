#!/bin/bash

# Directory where files will be created
TARGET_DIR="incoming"

# Number of files to create
NUM_FILES=1000

# Delay between file creations (in seconds, 0.5 = 500ms)
DELAY=0.5

# Ensure the target directory exists
mkdir -p "$TARGET_DIR"

echo "Starting to generate $NUM_FILES files in '$TARGET_DIR'..."

for i in $(seq 1 $NUM_FILES)
do
  # Generate a random filename
  FILENAME="testfile_$(date +%s%N)_$i.tmp"
  FILEPATH="$TARGET_DIR/$FILENAME"

  # Generate a random size between 1KB (1024 bytes) and 4KB (4096 bytes)
  # MIN_SIZE=1024, MAX_SIZE=4096
  # SIZE_BYTES=$(( RANDOM % (4096 - 1024 + 1) + 1024 ))
  # More robust way to get random number in a range for bash/sh
  MIN_KB=1
  MAX_KB=4
  RANDOM_KB=$(( RANDOM % (MAX_KB - MIN_KB + 1) + MIN_KB )) # Using $RANDOM for portability
  SIZE_BYTES=$(( RANDOM_KB * 1024 ))

  echo "Creating file $i/$NUM_FILES: $FILEPATH, Size: ${SIZE_BYTES}B"

  # Create file with random content using /dev/urandom
  # head -c ensures the exact byte size. dd can also be used.
  head -c "$SIZE_BYTES" /dev/urandom > "$FILEPATH"

  # Check if file creation was successful
  if [ $? -ne 0 ]; then
    echo "Error creating file: $FILEPATH. Exiting."
    exit 1
  fi

  # Wait for the specified delay
  sleep "$DELAY"
done

echo "Successfully generated $NUM_FILES files."
