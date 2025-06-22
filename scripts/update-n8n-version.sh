#!/bin/bash

# Script to fetch the latest n8n version from GitHub API
# This script fetches the latest release tag from the n8n GitHub repository

set -euo pipefail

# Function to get the latest version
get_latest_version() {
    # Fetch the latest release from GitHub API
    local latest_version
    latest_version=$(curl -s "https://api.github.com/repos/n8n-io/n8n/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | \
        sed 's/^n8n@//')
    
    echo "$latest_version"
}

# Function to update version file
update_version_file() {
    local new_version="$1"
    local version_file=".version"
    
    if [[ -f "$version_file" ]]; then
        local current_version
        current_version=$(cat "$version_file" | tr -d '\n')
        
        if [[ "$current_version" == "$new_version" ]]; then
            echo "Version is already up to date: $current_version"
            return 1
        fi
    fi
    
    echo "Updating version from ${current_version:-"unknown"} to $new_version"
    echo "$new_version" > "$version_file"
    return 0
}

# Main execution
main() {
    echo "Fetching latest n8n version..."
    
    local latest_version
    latest_version=$(get_latest_version)
    
    if [[ -z "$latest_version" ]]; then
        echo "Error: Could not fetch latest version"
        exit 1
    fi
    
    echo "Latest n8n version: $latest_version"
    
    if update_version_file "$latest_version"; then
        echo "Version file updated successfully"
        exit 0
    else
        echo "No update needed"
        exit 1
    fi
}

# Run main function
main "$@"