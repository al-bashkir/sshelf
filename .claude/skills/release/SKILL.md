---
name: release
description: Prepare a new sshelf release. Picks the version, adds the sshelf.spec changelog entry, runs the same checks as release CI plus packaging checks, then guides commit, PR, tag and release verification. Use when the user wants to prepare, cut or publish a new sshelf release or version.
argument-hint: "[X.Y.Z | patch | minor | major]"
---

# release

Prepare release `vX.Y.Z` of sshelf. In each snippet, replace `X.Y.Z` with the chosen version.

## Rules

- Follow AGENTS.md: commit only after the user says "commit", never commit on `main`, never run `git push`. The user pushes branches and tags.
- Tags `v*` are protected (no move, no delete) and releases are immutable. A pushed tag is final, and a published version number can never be used again. Every check must pass before the tag exists.
- Never run the built binary with the user's real config: always set `XDG_CONFIG_HOME` to a temp dir.
- No `Co-Authored-By` trailer in commits.
- If a step fails, stop, leave the changes uncommitted, and show the failing command and its output.

## 1. Preflight

```bash
set -e
case "$(git remote get-url origin)" in
  *al-bashkir/sshelf|*al-bashkir/sshelf.git) ;;
  *) echo "ERROR: origin is not al-bashkir/sshelf"; exit 1 ;;
esac
[ "$(git symbolic-ref --short HEAD)" = main ] || { echo "ERROR: switch to main first"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "ERROR: working tree not clean"; git status --short; exit 1; }
git fetch -q origin main --tags
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || { echo "ERROR: main differs from origin/main; pull first"; exit 1; }
gh auth status >/dev/null 2>&1 || { echo "ERROR: gh is not logged in"; exit 1; }
for t in go actionlint rpmspec; do command -v "$t" >/dev/null || { echo "ERROR: $t not found"; exit 1; }; done
echo "Preflight OK"
```

## 2. Pick the version

```bash
last=$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)
echo "last tag: ${last:-none}"
if [ -n "$last" ]; then git log --format='%h %s' "$last..HEAD"; else git log --format='%h %s'; fi
```

- `$ARGUMENTS` is `X.Y.Z` or `vX.Y.Z`: use it.
- `$ARGUMENTS` is `patch`, `minor` or `major`: bump the last tag.
- Empty: propose from the commits since the last tag. `!:` or `BREAKING CHANGE` gives major, any `feat` gives minor, else patch.
- No tag yet: ask the user for the version. Do not guess.
- With no commits since the last tag, stop: there is nothing to release.

Show the commit list and the version, and get the user's OK. Then check that the version is free:

```bash
V=X.Y.Z
git rev-parse -q --verify "refs/tags/v$V" >/dev/null && { echo "ERROR: local tag v$V exists"; exit 1; }
git ls-remote --exit-code --tags origin "refs/tags/v$V" >/dev/null && { echo "ERROR: tag v$V exists on origin"; exit 1; }
gh release view "v$V" >/dev/null 2>&1 && { echo "ERROR: release v$V exists"; exit 1; }
echo "v$V is free"
```

GitHub also blocks the name of a deleted immutable release. If the tag push is rejected later for that reason, use the next version.

## 3. Prepare the branch and files

```bash
V=X.Y.Z
git switch -c "release/v$V"
echo "* $(LC_ALL=C date '+%a %b %d %Y') $(git config user.name) <$(git config user.email)> - $V-1"
```

With the Edit tool:

1. `sshelf.spec`: add a new entry directly below the `%changelog` line. Use the header line printed above, then one `- ` bullet per user-visible change since the last tag, and a blank line before the old entries. Write plain sentences, for example "Do not quit the app when the search bar has focus". Skip chore, ci, test, style and docs-only commits unless users notice them. If the first entry is already for `X.Y.Z-1` (prepared earlier), do not add a second one: replace its header line with the one printed above and add any missing bullets.
2. `README.md`: set the example version in the local RPM build lines (`VERSION=... make -f .copr/Makefile srpm` and `sshelf-...-1.*.src.rpm`) to `X.Y.Z`.

Do not change `%global app_version`. The COPR Makefile writes the version there.

## 4. Checks

These are the release CI checks, plus a build, a workflow lint and a spec check.

```bash
set -e
V=X.Y.Z
tmp=$(mktemp -d)
go vet ./...
go test ./...
gv=$(grep -oE 'gosec@v[0-9.]+' .github/workflows/release.yml | cut -d@ -f2)
GOBIN="$tmp/bin" go install "github.com/securego/gosec/v2/cmd/gosec@$gv"
"$tmp/bin/gosec" -quiet ./...
go build -o build/sshelf ./cmd/sshelf
XDG_CONFIG_HOME="$tmp/xdg" ./build/sshelf completion bash >/dev/null
actionlint
sed "s/^%global app_version.*/%global app_version $V/" sshelf.spec > "$tmp/sshelf.spec"
[ "$(rpmspec -q --srpm --qf '%{version}' "$tmp/sshelf.spec")" = "$V" ] || { echo "ERROR: spec version is not $V"; exit 1; }
sed -n '/^%changelog/{n;p;q}' sshelf.spec | grep -q -- " - $V-1\$" || { echo "ERROR: first changelog entry is not $V-1"; exit 1; }
echo "Checks OK (gosec $gv)"
```

## 5. Stop and report

Show the version, the new changelog entry and `git diff --stat`. Ask the user to say "commit". Do nothing more until then.

## 6. After "commit"

```bash
V=X.Y.Z
git add sshelf.spec README.md
git commit -m "chore(release): prepare v$V"
```

Give the user these commands. `main` takes changes only through a PR:

```bash
git push -u origin release/vX.Y.Z
gh pr create --fill --base main
gh pr merge --squash --delete-branch
```

Ask the user to say when the PR is merged.

## 7. After the merge: tag

```bash
set -e
V=X.Y.Z
git switch main
git pull --ff-only
git log -1 --format='%h %s'
git diff --quiet "release/v$V" HEAD || echo "WARNING: main differs from the tested branch; run step 4 again before tagging"
```

`HEAD` must be the release commit (`chore(release): prepare vX.Y.Z (#N)`). If `main` differs from the tested branch, run step 4 again. Then create the annotated tag:

```bash
V=X.Y.Z
git tag -a "v$V" -m "sshelf v$V"
git branch -d "release/v$V"
```

Tell the user that the next command is final and publishes the release:

```bash
git push origin vX.Y.Z
```

## 8. After the tag push: verify

Find and watch the release run:

```bash
gh run list --workflow release.yml --limit 1
gh run watch <run-id> --exit-status
```

Check that all 10 assets exist:

```bash
V=X.Y.Z
have=$(gh release view "v$V" --json assets --jq '.assets[].name')
missing=0
for p in linux_amd64.tar.gz linux_arm64.tar.gz darwin_amd64.tar.gz darwin_arm64.tar.gz windows_amd64.zip; do
  for f in "sshelf_v${V}_$p" "sshelf_v${V}_$p.sha256"; do
    printf '%s\n' "$have" | grep -qx "$f" || { echo "MISSING $f"; missing=1; }
  done
done
[ "$missing" = 0 ] && echo "All 10 assets present"
```

On a linux/amd64 host, check one archive and run it:

```bash
V=X.Y.Z
tmp=$(mktemp -d); mkdir "$tmp/dist"
gh release download "v$V" -p "sshelf_v${V}_linux_amd64.tar.gz*" -D "$tmp/dist"
(cd "$tmp" && sha256sum -c "dist/sshelf_v${V}_linux_amd64.tar.gz.sha256")
tar -xzf "$tmp/dist/sshelf_v${V}_linux_amd64.tar.gz" -C "$tmp"
XDG_CONFIG_HOME="$tmp/xdg" "$tmp/sshelf" -h 2>&1 | head -1
```

If the run fails before the release is published, the tag stays where it is and cannot move. Fix the problem on `main` through a PR and release the next patch version. Do not try to delete the tag.

Finish with the release URL: `https://github.com/al-bashkir/sshelf/releases/tag/vX.Y.Z`.
