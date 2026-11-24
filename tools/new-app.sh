#!/usr/bin/env bash
# Start a new Go service product from this repo's full layered skeleton.
#
#   tools/new-app.sh billing ~/Documents/projects/billing
#   MODULE=github.com/acme/billing tools/new-app.sh billing ~/src/billing
#
# Copies the committed tree (git archive HEAD), so the new product keeps AGENTS.md,
# CLAUDE.md, tools/ (scaffolding + hook + arch-check), the Jenkinsfile and the arch-qube
# gate. Copying a few packages into a fresh `go mod init` loses all of that (AGENTS.md §1).
set -euo pipefail

[ $# -eq 2 ] || { echo "usage: [MODULE=<go module path>] $0 <app-name> <dest-dir>" >&2; exit 1; }
APP="$1"
DEST="$2"
[[ "$APP" =~ ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ ]] || { echo "app-name: lower-case letters/digits, words joined by '-' (it becomes the binary, image, Sonar key and CI names)" >&2; exit 1; }
[[ ! "$APP" =~ ^(arcana|arcana-cloud|arcana-cloud-go|go-app)$ ]] || { echo "app-name must differ from the template's own names" >&2; exit 1; }
[ ! -e "$DEST" ] || { echo "$DEST already exists — refusing to overwrite" >&2; exit 1; }
command -v perl >/dev/null || { echo "perl is required" >&2; exit 1; }

OLD_MODULE="github.com/jrjohn/arcana-cloud-go"
MODULE="${MODULE:-github.com/jrjohn/${APP}}"
[[ "$MODULE" =~ ^[a-z0-9.-]+(/[A-Za-z0-9._-]+)+$ ]] || { echo "MODULE '$MODULE' is not a valid Go module path" >&2; exit 1; }
APP_US="$(printf '%s' "$APP" | tr '-' '_')"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[ "$(awk '/^module /{print $2; exit}' "$ROOT/go.mod")" = "$OLD_MODULE" ] || { echo "new-app: go.mod module is not $OLD_MODULE — run this from the template, not from a product" >&2; exit 1; }
git -C "$ROOT" diff --quiet HEAD -- AGENTS.md tools/ || echo "note: uncommitted changes in AGENTS.md/tools are NOT copied (git archive HEAD)" >&2

mkdir -p "$DEST"
git -C "$ROOT" archive HEAD | tar -x -C "$DEST"
SRC_SHA="$(git -C "$ROOT" rev-parse --short HEAD)"
cd "$DEST"

# ---- Product identity. Every place the template name is load-bearing:
#  1. Go module path (go.mod + every import)              github.com/jrjohn/arcana-cloud-go -> $MODULE
#  2. binary / app / k8s / compose names                   arcana-cloud-go, arcana-cloud-*, arcana_cloud
#  3. Jenkins APP_NAME + image (go-app), Sonar key/name     go-app, "Go App"
#  4. per-host CI names — two products on one Jenkins host must not share them:
#     compose project/network/containers arcana-ci-go-*, kind clusters + namespace arcana-ci-*
#     (the Jenkins post step and kind-smoke-test.sh delete every cluster matching the prefix),
#     arch-qube container arcana-arch-qube-go-*, coverage container go-cov-tmp-*, lock ci-go-layered,
#     test containers/networks arcana-*-test in docker-compose*.yml.
# Not renamed on purpose: the ARCANA_ env prefix (viper config contract), arcana_* metric
# names, the local test DB name arcana_test, and arcana.boo (registry / arch-qube image host).
# tools/ is excluded: these scripts describe the template and must keep working in the product.
ALL="$(grep -rIl --exclude-dir=.git --exclude-dir=tools --exclude=CHANGELOG.md -e arcana -e go-app -e 'Go App' -e go-cov -e ci-go- . || true)"
[ -n "$ALL" ] || { echo "new-app: nothing to rename?" >&2; exit 1; }
printf '%s\n' "$ALL" | xargs perl -pi -e "
    s#\Q${OLD_MODULE}\E#${MODULE}#g;
    s/arcana-cloud-go(?![\w-])/${APP}/g;
    s/(?<![\w.\/-])arcana-cloud(?!-(springboot|rust|python|nodejs|java))(?![\w])/${APP}/g;
    s/(?<![\w])arcana_cloud(?![\w])/${APP_US}/g;
    s/\bgo-app\b/${APP}/g;
    s/Go App/${APP}/g;
    s/\bgo-cov-tmp-/${APP}-cov-tmp-/g;
    s/\bci-go-layered\b/ci-${APP}-layered/g;
"
# CI / container infrastructure files: every remaining arcana-<x> name is a host-level resource.
INFRA="$(printf '%s\n' "$ALL" | grep -E '(^\./Jenkinsfile$|^\./Makefile$|docker-compose[^/]*\.ya?ml$|^\./deployment/|^\./scripts/)' || true)"
[ -z "$INFRA" ] || printf '%s\n' "$INFRA" | xargs perl -pi -e "s/(?<![\w.\/-])arcana-(?!cloud-(springboot|rust|python|nodejs|java)|ios\b)(?=[a-z])/${APP}-/g"

# Fail loudly if a load-bearing template identity survived.
LEFT="$(grep -rIn --exclude-dir=.git --exclude-dir=tools --exclude=CHANGELOG.md -E "jrjohn/arcana-cloud-go|arcana-cloud-go|\\bgo-app\\b|arcana-ci|arcana-arch-qube-go|ci-go-layered" . || true)"
if [ -n "$LEFT" ]; then
    echo "new-app: template identities still present after rename:" >&2
    printf '%s\n' "$LEFT" | head >&2
    exit 1
fi
[ "$(awk '/^module /{print $2; exit}' go.mod)" = "$MODULE" ] || { echo "new-app: go.mod module was not renamed" >&2; exit 1; }

# ---- Fresh history: version 0.1.0, empty changelog.
rm -f CHANGELOG.md
printf '{\n  ".": "0.1.0"\n}\n' > .release-please-manifest.json

cat >> AGENTS.md <<EOT

## Origin

Created from arcana-cloud-go @ \`${SRC_SHA}\` by \`tools/new-app.sh\` (module
\`${MODULE}\`, app name \`${APP}\`). The layers, rules and definition of done above apply
unchanged. Remove template features you do not need through \`internal/di/\`, not by flattening.
EOT

git init -q
tools/install-hooks.sh >/dev/null
git add -A
# --no-verify only for this import commit: every file counts as "added" here, so the hook
# would judge the whole template (including legacy files) as new code. The template itself
# passes arch-check; every later commit in the product goes through the hook.
git -c user.name="${GIT_AUTHOR_NAME:-new-app}" -c user.email="${GIT_AUTHOR_EMAIL:-new-app@localhost}" \
    commit -q --no-verify -m "chore: start ${APP} from arcana-cloud-go ${SRC_SHA}"

echo "created $DEST (${APP}, module ${MODULE}) from arcana-cloud-go ${SRC_SHA}; hooks: $(git config core.hooksPath)"
echo "next: cd $DEST && tools/new-feature.sh <Name>   # one vertical slice per feature"
