#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
gcldir="$workspace/src/github.com/gclchaineum"
if [ ! -L "$gcldir/go-gclchaineum" ]; then
    mkdir -p "$gcldir"
    cd "$gcldir"
    ln -s ../../../../../. go-gclchaineum
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$gcldir/go-gclchaineum"
PWD="$gcldir/go-gclchaineum"

# Launch the arguments with the configured environment.
exec "$@"
