#!/bin/bash

# Run the table test converter on a directory
# Usage: ./run_conversion.sh <directory_path>

if [ $# -ne 1 ]; then
    echo "Usage: ./run_conversion.sh <directory_path>"
    exit 1
fi

DIR_PATH=$1

# Compile the converter
echo "Compiling table test converter..."
go build -o table_converter tabletests.go

if [ $? -ne 0 ]; then
    echo "Failed to compile the converter"
    exit 1
fi

# Run the converter
echo "Running conversion on $DIR_PATH..."
./table_converter "$DIR_PATH"

# Cleanup
rm table_converter

echo "Done!"