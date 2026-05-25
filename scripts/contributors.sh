#!/bin/bash

echo "| Contributor | Contributions |"
echo "|-------------|--------------|"

gh api repos/$GITHUB_REPOSITORY/contributors --paginate \
| jq -r '.[] | "| \(.login) | \(.contributions) |"'