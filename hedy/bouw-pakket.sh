#!/usr/bin/env bash
# Bouwt het Hedy-pakket voor Unscroll. Draait in Debian 12 (zoals de apparaten) in GitHub Actions.
# Gebruik: bash hedy/bouw-pakket.sh UITMAP
set -euxo pipefail
HEDYLANG=0.0.9                                     # https://pypi.org/project/hedylang/
HEDY_COMMIT=38d5db7214ff9db8e0ee5e009e97a554897f2e53  # opdrachten en vertalingen van hedy.org
HIER="$(cd "$(dirname "$0")" && pwd)"
UIT="$1"
W="$(mktemp -d)"
B="$W/pakket"

apt-get update
apt-get install -y --no-install-recommends python3 python3-pip python3-regex python3-yaml git ca-certificates

mkdir -p "$B/lib"
# hedylang vraagt officieel 3.12; pas_hedy_aan.py maakt het geschikt voor 3.11
pip3 download --no-deps --only-binary=:all: --python-version 3.12 -d "$W/whl" "hedylang==$HEDYLANG"
python3 -m zipfile -e "$W"/whl/hedylang-*.whl "$W/x"
cp -r "$W/x/hedy" "$B/lib/"
python3 "$HIER/pas_hedy_aan.py" "$B/lib/hedy"
# pure Python; regex komt uit Debian (python3-regex)
pip3 install --break-system-packages --no-deps --target "$B/lib" "lark==1.1.9" "ruamel.yaml==0.18.6" unidecode iso-639 iso3166

git clone -q --filter=blob:none --no-checkout https://github.com/hedyorg/hedy "$W/site"
git -C "$W/site" checkout "$HEDY_COMMIT" -- content/adventures/nl.yaml content/adventures/en.yaml website_content.py \
  translations/nl/LC_MESSAGES/messages.po translations/en/LC_MESSAGES/messages.po LICENSE
python3 "$HIER/maak_inhoud.py" "$W/site" "$B/lib/hedy" "$B"
cp "$W/site/LICENSE" "$B/HEDY-LICENSE"
cp -r "$HIER/unscroll_hedy.py" "$HIER/schildpad" "$B/"

# zelftest: een paar programma's op verschillende niveaus, in het Nederlands en Engels
toets() {  # toets NIVEAU TAAL VERWACHT CODE [INVOER]
  printf '%s\n' "$4" > "$W/p.hedy"
  uitvoer="$(printf '%s\n' "${5:-}" | python3 -I "$B/unscroll_hedy.py" "$1" "$2" "$W/p.hedy" 2>&1 || true)"
  echo "$uitvoer"
  grep -q -- "$3" <<<"$uitvoer"
}
toets 1 nl "hallo wereld" "print hallo wereld"
toets 1 nl "Sam" "vraag hoe heet jij?
echo hallo" "Sam"
toets 2 en "pizza" "food is pizza
print I like food"
toets 4 en "dice" "choices is 1, 2, 3
print 'dice ' choices at random"
toets 5 nl "goed" "naam is Sam
als naam is Sam print 'goed' anders print 'fout'"
toets 7 en "hoi" "repeat 2 times print 'hoi'"
toets 1 nl "⠀\|⢸\|⣀" "vooruit 50
draai rechts
vooruit 50"
toets 1 nl "HEDYFOUT" "prnt hallo"
toets 12 en "HEDYFOUT" "import os"

mkdir -p "$UIT"
tar -czf "$UIT/unscroll-hedy.tar.gz" -C "$B" .
(cd "$UIT" && sha256sum unscroll-hedy.tar.gz > unscroll-hedy.tar.gz.sha256)
du -sh "$UIT/unscroll-hedy.tar.gz"
