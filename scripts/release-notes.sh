#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi

current_tag="$1"
current_commit="$(git rev-list -n 1 "${current_tag}")"
previous_tag="$(git describe --tags --abbrev=0 "${current_commit}^" 2>/dev/null || true)"

if [[ -n "${previous_tag}" ]]; then
  previous_commit="$(git rev-list -n 1 "${previous_tag}")"
  range="${previous_commit}..${current_commit}"
  title="Release ${current_tag} (${previous_tag}...${current_tag})"
else
  range="${current_commit}"
  title="Release ${current_tag}"
fi

{
  echo "# ${title}"
  echo
  if [[ -n "${previous_tag}" ]]; then
    echo "Changes since ${previous_tag}:"
  else
    echo "Changes included in this first tagged release:"
  fi
  echo
  git log --no-merges --pretty=format:"- %s (%h)" "${range}"
  echo
}
