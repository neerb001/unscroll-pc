#!/usr/bin/env bash
# Installeert de kinderkamer-terminal op een verse Debian 12 (32-bit of 64-bit).
# Gebruik (als root):  bash install.sh
# Opnieuw draaien is veilig: het script overschrijft alleen zijn eigen bestanden.
set -euo pipefail

KIND="${KIND:-kind}"                 # gebruikersnaam van het kinderaccount
REPO="${REPO:-OWNER/REPO}"           # GitHub-repository voor updates
HIER="$(cd "$(dirname "$0")" && pwd)"

[[ $EUID -eq 0 ]] || { echo "Draai dit script als root (sudo bash install.sh)."; exit 1; }
. /etc/os-release
[[ "${VERSION_ID:-}" == "12" ]] || echo "Let op: getest op Debian 12, jij hebt ${PRETTY_NAME:-onbekend}."

echo "==> Pakketten installeren"
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  cage foot fonts-jetbrains-mono \
  network-manager alsa-utils mpv \
  curl jq ca-certificates zram-tools \
  avahi-daemon libnss-mdns \
  sudo

echo "==> Geheugen sparen met zram"
sed -i 's/^#\?PERCENT=.*/PERCENT=50/' /etc/default/zramswap || true
systemctl enable --now zramswap.service || true

echo "==> Kinderaccount '$KIND' (zonder beheerrechten)"
if ! id "$KIND" &>/dev/null; then
  adduser --disabled-password --gecos "Kinderkamer" "$KIND"
fi
for g in audio video netdev render input; do getent group "$g" >/dev/null && usermod -aG "$g" "$KIND"; done
gpasswd -d "$KIND" sudo 2>/dev/null || true

echo "==> Bestanden plaatsen"
install -d -m 0755 /etc/eee /opt/eee/releases
install -m 0755 "$HIER/scripts/eee-sessie" /usr/local/bin/eee-sessie
install -m 0755 "$HIER/scripts/eee-update" /usr/local/sbin/eee-update
[[ -f /etc/eee/ouders.yaml ]] || install -m 0644 "$HIER/config/ouders.example.yaml" /etc/eee/ouders.yaml
[[ -f /etc/eee/update.conf ]] || sed "s#OWNER/REPO#$REPO#" "$HIER/config/update.example.conf" > /etc/eee/update.conf
[[ -f /etc/eee/geheimen.env ]] || { install -m 0640 -g "$KIND" "$HIER/config/geheimen.example.env" /etc/eee/geheimen.env; }
chown root:root /etc/eee/ouders.yaml /etc/eee/update.conf
chmod 0644 /etc/eee/ouders.yaml /etc/eee/update.conf

install -d -o "$KIND" -g "$KIND" "/home/$KIND/.config/foot"
install -m 0644 -o "$KIND" -g "$KIND" "$HIER/config/foot.ini" "/home/$KIND/.config/foot/foot.ini"

echo "==> App installeren"
if [[ -x "$HIER/eee" ]]; then
  # handmatig meegeleverde binary (bijvoorbeeld zelf gebouwd)
  install -d /opt/eee/releases/lokaal
  install -m 0755 "$HIER/eee" /opt/eee/releases/lokaal/eee
  ln -sfn /opt/eee/releases/lokaal /opt/eee/huidig
else
  /usr/local/sbin/eee-update --nu || echo "Nog geen release gevonden; de updater probeert het later opnieuw."
fi
ln -sfn /opt/eee/huidig/eee /usr/local/bin/eee

echo "==> Automatisch inloggen op tty1 en direct de app starten"
install -d /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/kinderkamer.conf <<EOF
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin $KIND --noclear %I \$TERM
EOF
cat > "/home/$KIND/.bash_profile" <<'EOF'
# Start de kinderkamer-terminal op het scherm (alleen op tty1).
if [[ "$(tty)" == "/dev/tty1" ]]; then
  exec /usr/local/bin/eee-sessie
fi
EOF
chown "$KIND:$KIND" "/home/$KIND/.bash_profile"

echo "==> Updater elke 15 minuten"
install -m 0644 "$HIER/systemd/eee-update.service" /etc/systemd/system/
install -m 0644 "$HIER/systemd/eee-update.timer" /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now eee-update.timer

echo "==> Volume beperken en bewaren"
amixer -q set Master 40% unmute 2>/dev/null || true
alsactl store 2>/dev/null || true

cat <<EOF

Klaar. Nog drie dingen:
  1. Stel een pincode in voor de ouders:
       /usr/local/bin/eee --pin-hash 1234
     en zet de uitkomst bij pin_sha256 in /etc/eee/ouders.yaml
  2. Pas de avondslot-tijden aan in /etc/eee/ouders.yaml
  3. Herstart: sudo reboot  (daarna start de app vanzelf)

Wifi instellen kan met: sudo nmtui
EOF
