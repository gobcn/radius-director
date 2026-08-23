#!/bin/sh
set -eu

TENANT_SOURCE="/opt/radius-director/tenant"
FREERADIUS_CONFIG="/etc/freeradius"
MANIFEST="${TENANT_SOURCE}/.radius-director/manifest.yaml"

echo "RADIUS Director: preparing tenant configuration"

if [ ! -d "$TENANT_SOURCE" ]; then
    echo "ERROR: tenant configuration directory does not exist: $TENANT_SOURCE" >&2
    exit 1
fi

if [ ! -f "$MANIFEST" ]; then
    echo "ERROR: tenant manifest does not exist: $MANIFEST" >&2
    exit 1
fi

echo "RADIUS Director: copying tenant configuration"

cp -a "$TENANT_SOURCE/." "$FREERADIUS_CONFIG/"
rm -rf "$FREERADIUS_CONFIG/.radius-director"

echo "RADIUS Director: applying removal manifest"

# Process the simple YAML structure produced by RADIUS Director:
#
# remove:
#     - path/to/file
#     - another/path
#
# We deliberately do not use a general-purpose YAML parser here. The
# generated manifest has a tightly controlled format.

awk '
    /^remove:[[:space:]]*$/ {
        in_remove = 1
        next
    }

    in_remove && /^[[:space:]]*-[[:space:]]*/ {
        path = $0
        sub(/^[[:space:]]*-[[:space:]]*/, "", path)
        print path
        next
    }

    in_remove {
        exit
    }
' "$MANIFEST" |
while IFS= read -r relative_path; do

    # Ignore an empty removal entry.
    [ -n "$relative_path" ] || continue

    # The manifest is allowed to contain paths relative to /etc/freeradius.
    #
    # Never permit an absolute path or path traversal.
    case "$relative_path" in
        /*)
            echo "ERROR: manifest contains absolute path: $relative_path" >&2
            exit 1
            ;;
        ../*|*/../*|..)
            echo "ERROR: manifest contains path traversal: $relative_path" >&2
            exit 1
            ;;
    esac

    target="${FREERADIUS_CONFIG}/${relative_path}"

    echo "RADIUS Director: removing ${relative_path}"

    rm -rf -- "$target"
done

echo "RADIUS Director: starting FreeRADIUS"

exec freeradius -fl stdout