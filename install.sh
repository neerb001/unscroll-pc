#!/usr/bin/env bash
# Installeert Unscroll op een verse Debian 12 (32-bit of 64-bit).
# Gebruik (als root):  bash install.sh
# Opnieuw draaien is veilig: het script overschrijft alleen zijn eigen bestanden.
set -euo pipefail

KIND="${KIND:-kind}"                 # gebruikersnaam van het kinderaccount
REPO="${REPO:-neerb001/unscroll-pc}"           # GitHub-repository voor updates
HIER="$(cd "$(dirname "$0")" && pwd)"

[[ $EUID -eq 0 ]] || { echo "Draai dit script als root (sudo bash install.sh)."; exit 1; }
# shellcheck source=/dev/null
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
  adduser --disabled-password --gecos "Unscroll" "$KIND"
fi
for g in audio video netdev render input; do getent group "$g" >/dev/null && usermod -aG "$g" "$KIND"; done
gpasswd -d "$KIND" sudo 2>/dev/null || true

echo "==> Bestanden plaatsen"
install -d -m 0755 /etc/unscroll /opt/unscroll/releases
install -m 0755 "$HIER/scripts/unscroll-sessie" /usr/local/bin/unscroll-sessie
install -m 0755 "$HIER/scripts/unscroll-update" /usr/local/sbin/unscroll-update
install -m 0755 "$HIER/scripts/unscroll-ouder" /usr/local/sbin/unscroll-ouder
[[ -f /etc/unscroll/ouders.yaml ]] || install -m 0644 "$HIER/config/ouders.example.yaml" /etc/unscroll/ouders.yaml
[[ -f /etc/unscroll/update.conf ]] || sed "s#^REPO=.*#REPO=\"$REPO\"#" "$HIER/config/update.example.conf" > /etc/unscroll/update.conf
[[ -f /etc/unscroll/geheimen.env ]] || { install -m 0640 -g "$KIND" "$HIER/config/geheimen.example.env" /etc/unscroll/geheimen.env; }
chown root:root /etc/unscroll/ouders.yaml /etc/unscroll/update.conf
chmod 0644 /etc/unscroll/ouders.yaml /etc/unscroll/update.conf

install -d -o "$KIND" -g "$KIND" "/home/$KIND/.config/foot"
install -m 0644 -o "$KIND" -g "$KIND" "$HIER/config/foot.ini" "/home/$KIND/.config/foot/foot.ini"

echo "==> App installeren"
if [[ -x "$HIER/unscroll" ]]; then
  # handmatig meegeleverde binary (bijvoorbeeld zelf gebouwd)
  install -d /opt/unscroll/releases/lokaal
  install -m 0755 "$HIER/unscroll" /opt/unscroll/releases/lokaal/unscroll
  ln -sfn /opt/unscroll/releases/lokaal /opt/unscroll/huidig
else
  /usr/local/sbin/unscroll-update --nu || echo "Nog geen release gevonden; de updater probeert het later opnieuw."
fi
ln -sfn /opt/unscroll/huidig/unscroll /usr/local/bin/unscroll

echo "==> Automatisch inloggen op tty1 en direct de app starten"
install -d /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/kinderkamer.conf <<EOF
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin $KIND --noclear %I \$TERM
EOF
cat > "/home/$KIND/.bash_profile" <<'EOF'
# Start Unscroll op het scherm (alleen op tty1).
if [[ "$(tty)" == "/dev/tty1" ]]; then
  exec /usr/local/bin/unscroll-sessie
fi
EOF
chown "$KIND:$KIND" "/home/$KIND/.bash_profile"

echo "==> Updater elke 15 minuten"
install -m 0644 "$HIER/systemd/unscroll-update.service" /etc/systemd/system/
install -m 0644 "$HIER/systemd/unscroll-update.timer" /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now unscroll-update.timer

echo "==> Volume beperken en bewaren"
amixer -q set Master 40% unmute 2>/dev/null || true
alsactl store 2>/dev/null || true

cat <<EOF

Klaar. Stel nu lokaal de pincode, het avondslot en de geheimen in met de ouderhulp:

  sudo unscroll-ouder

Wachtwoorden en sleutels komen alleen in /etc/unscroll op dit apparaat, nooit in git.
Wifi instellen kan met: sudo nmtui
Daarna herstarten: sudo reboot
EOF
