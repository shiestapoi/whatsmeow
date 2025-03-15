#!/bin/bash

# This script directly fixes the problematic file
# The error occurs in file_waChatLockSettings_WAProtobufsChatLockSettings_proto_init() function

TARGET_FILE="./proto/waChatLockSettings/WAProtobufsChatLockSettings.pb.go"

if [ -f "$TARGET_FILE" ]; then
  echo "Fixing $TARGET_FILE..."

  # Create a backup
  cp "$TARGET_FILE" "${TARGET_FILE}.bak"

  # Fix the slice bounds issue by finding the unmarshalSeed function call in the init function
  # and modifying any problematic array accesses like x[-1]
  sed -i 's/x\[-1\]/x\[0\]/g' "$TARGET_FILE"

  echo "Fix applied. Original file backed up as ${TARGET_FILE}.bak"
else
  echo "Target file not found: $TARGET_FILE"
fi
