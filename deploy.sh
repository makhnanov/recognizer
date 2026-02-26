#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Git Release Tagger ===${NC}\n"

# Check if we are in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: This is not a git repository!${NC}"
    exit 1
fi

# Get the last tag
LAST_TAG=$(git tag --sort=-v:refname | head -1)

if [ -z "$LAST_TAG" ]; then
    echo -e "${YELLOW}No tags found. Using version 0.0.0${NC}"
    LAST_TAG="0.0.0"
fi

echo -e "Current version: ${GREEN}${LAST_TAG}${NC}\n"

# Parse version and detect prefix
PREFIX=""
if [[ $LAST_TAG =~ ^v ]]; then
    PREFIX="v"
fi

if [[ $LAST_TAG =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
else
    echo -e "${RED}Error: Invalid tag format '$LAST_TAG'${NC}"
    echo "Expected format: X.Y.Z or vX.Y.Z"
    exit 1
fi

# Calculate new versions (preserving v prefix if present)
NEW_MAJOR="${PREFIX}$((MAJOR + 1)).0.0"
NEW_MINOR="${PREFIX}${MAJOR}.$((MINOR + 1)).0"
NEW_PATCH="${PREFIX}${MAJOR}.${MINOR}.$((PATCH + 1))"

# Get the last commit message
LAST_COMMIT_MSG=$(git log -1 --pretty=%B)
LAST_COMMIT_HASH=$(git log -1 --pretty=%h)

echo -e "Last commit: ${YELLOW}${LAST_COMMIT_HASH}${NC}"
echo -e "Message: ${YELLOW}${LAST_COMMIT_MSG}${NC}\n"

# Interactive selection via dialog
TEMP_FILE=$(mktemp)

dialog --clear \
    --title "Git Release Tagger" \
    --default-item "1" \
    --menu "Select version update type:" 15 70 4 \
    "1" "PATCH ${LAST_TAG} → ${NEW_PATCH} (bug fixes)" \
    "2" "MINOR ${LAST_TAG} → ${NEW_MINOR} (new features)" \
    "3" "MAJOR ${LAST_TAG} → ${NEW_MAJOR} (breaking changes)" \
    "4" "Cancel" 2> "$TEMP_FILE"

choice=$?
selected=$(cat "$TEMP_FILE")
rm -f "$TEMP_FILE"

# Clear screen after dialog
clear

# If ESC or Cancel was pressed
if [ $choice -ne 0 ]; then
    echo -e "${YELLOW}Cancelled by user${NC}"
    exit 0
fi

case $selected in
    1)
        NEW_VERSION="$NEW_PATCH"
        VERSION_TYPE="PATCH"
        ;;
    2)
        NEW_VERSION="$NEW_MINOR"
        VERSION_TYPE="MINOR"
        ;;
    3)
        NEW_VERSION="$NEW_MAJOR"
        VERSION_TYPE="MAJOR"
        ;;
    4)
        echo -e "${YELLOW}Cancelled by user${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}Creating tag: ${NEW_VERSION}${NC}"
echo -e "Update type: ${BLUE}${VERSION_TYPE}${NC}\n"

# Confirmation
read -p "Continue? (Y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[NnНн]$ ]]; then
    echo -e "${YELLOW}Cancelled by user${NC}"
    exit 0
fi

# Create tag with message from last commit
echo -e "\n${BLUE}Creating tag...${NC}"
git tag -a "$NEW_VERSION" -m "$LAST_COMMIT_MSG"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Tag ${NEW_VERSION} created${NC}"
else
    echo -e "${RED}✗ Error creating tag${NC}"
    exit 1
fi

# Push tag
echo -e "\n${BLUE}Publishing tag to repository...${NC}"
git push origin "$NEW_VERSION"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Tag ${NEW_VERSION} published${NC}"
    echo -e "\n${GREEN}=== Release successfully created! ===${NC}"
    echo -e "Version: ${GREEN}${NEW_VERSION}${NC}"
    echo -e "Type: ${BLUE}${VERSION_TYPE}${NC}"
else
    echo -e "${RED}✗ Error publishing tag${NC}"
    echo -e "${YELLOW}Tag created locally but not published${NC}"
    echo -e "You can publish it manually: ${YELLOW}git push origin ${NEW_VERSION}${NC}"
    exit 1
fi
