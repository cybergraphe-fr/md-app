#!/usr/bin/env bash
# Sign a Windows EXE using osslsigncode in Docker.
# Usage: scripts/sign-exe.sh <input.exe> [output.exe]
#
# Requires: secrets/codesign/codesign.crt and secrets/codesign/codesign.key
# These are self-signed; replace with a CA-issued code signing cert for SmartScreen trust.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CERT_DIR="$PROJECT_ROOT/secrets/codesign"

INPUT="${1:?Usage: sign-exe.sh <input.exe> [output.exe]}"
OUTPUT="${2:-${INPUT%.exe}-signed.exe}"

if [[ ! -f "$CERT_DIR/codesign.crt" || ! -f "$CERT_DIR/codesign.key" ]]; then
  echo "ERROR: Missing signing certificate in $CERT_DIR" >&2
  echo "Generate one with:" >&2
  echo "  openssl req -x509 -newkey rsa:4096 -keyout codesign.key -out codesign.crt \\" >&2
  echo "    -days 3650 -nodes -subj '/CN=Cybergraphe/O=Cybergraphe/C=FR' \\" >&2
  echo "    -addext 'extendedKeyUsage=codeSigning' -addext 'keyUsage=digitalSignature'" >&2
  exit 1
fi

INPUT_ABS="$(cd "$(dirname "$INPUT")" && pwd)/$(basename "$INPUT")"
OUTPUT_ABS="$(cd "$(dirname "$OUTPUT")" && pwd)/$(basename "$OUTPUT")"
INPUT_DIR="$(dirname "$INPUT_ABS")"
OUTPUT_DIR="$(dirname "$OUTPUT_ABS")"

echo "Signing $INPUT -> $OUTPUT ..."

MOUNTS="-v $INPUT_DIR:/input:ro -v $CERT_DIR:/certs:ro"
if [[ "$INPUT_DIR" != "$OUTPUT_DIR" ]]; then
  MOUNTS="$MOUNTS -v $OUTPUT_DIR:/output"
  OUT_PATH="/output/$(basename "$OUTPUT_ABS")"
else
  MOUNTS="-v $INPUT_DIR:/input -v $CERT_DIR:/certs:ro"
  OUT_PATH="/input/$(basename "$OUTPUT_ABS")"
fi

docker run --rm $MOUNTS debian:bookworm-slim sh -c "
  apt-get update -qq && apt-get install -y -qq osslsigncode > /dev/null 2>&1 &&
  osslsigncode sign \
    -certs /certs/codesign.crt \
    -key /certs/codesign.key \
    -n 'MD by Cybergraphe' \
    -i 'https://md.cybergraphe.fr' \
    -t http://timestamp.digicert.com \
    -in '/input/$(basename "$INPUT_ABS")' \
    -out '$OUT_PATH'
"

echo "Signed: $OUTPUT"
