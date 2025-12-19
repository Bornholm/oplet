#!/bin/sh
set -e

# Oplet places "file" type inputs in this directory
SOURCE_IMAGE="/oplet/inputs/SOURCE_IMAGE"

# Retrieving environment variables (defined via Oplet Labels)
QUALITY="${IMG_QUALITY:-85}"   # Default value 85
TARGET_WIDTH="${IMG_WIDTH}"    # Target width
GRAYSCALE="${TO_GRAYSCALE}"    # "true" or "false"

# Oplet will automatically collect all files placed in /oplet/outputs
output_path="/oplet/outputs/optimized.jpg"

echo "🚀 Starting optimization for: $FILE_NAME"
echo "ℹ️  Parameters: Quality=$QUALITY, Width=$TARGET_WIDTH, Grayscale=$GRAYSCALE"

# Building the magick command
CMD="magick '$SOURCE_IMAGE' -quality $QUALITY"

if [ -n "$TARGET_WIDTH" ]; then
    CMD="$CMD -resize ${TARGET_WIDTH}x"
fi

if [ "$GRAYSCALE" = "true" ]; then
    CMD="$CMD -colorspace Gray"
fi

CMD="$CMD '$output_path'"

# Execution
eval $CMD

echo "✅ Success! Image optimized: $output_path"
ls -lh "$output_path"