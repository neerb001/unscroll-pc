#!/usr/bin/env bash
# Installs Unscroll on a fresh Debian 12 (32-bit or 64-bit).
# Usage (as root):  bash install.sh
# Safe to run again: it only overwrites its own files.
set -euo pipefail

KIND="${KIND:-kind}"                 # user name of the child account
REPO="${REPO:-neerb001/unscroll-pc}"           # GitHub repository for updates
HIER="$(cd "$(dirname "$0")" && pwd)"

[[ $EUID -eq 0 ]] || { echo "Run this script as root (sudo bash install.sh)."; exit 1; }
# shellcheck source=/dev/null
. /etc/os-release
[[ "${VERSION_ID:-}" == "12" ]] || echo "Note: tested on Debian 12, you have ${PRETTY_NAME:-unknown}."

echo "==> Installing packages"
# Offline installation: if a "debs" folder with a package index sits next to this script
# (see the offline bundle), apt uses only that folder and needs no internet.
APT=(apt-get)
OFFLINE_LIST=""
if [[ -f "$HIER/debs/Packages.gz" ]]; then
  echo "    offline bundle found, not using the internet"
  OFFLINE_LIST=/etc/apt/sources.list.d/unscroll-offline.list
  echo "deb [trusted=yes] file:$HIER/debs ./" > "$OFFLINE_LIST"
  APT=(apt-get -o Dir::Etc::sourcelist="$OFFLINE_LIST" -o Dir::Etc::sourceparts=- -o APT::Get::List-Cleanup=0)
fi
"${APT[@]}" update
DEBIAN_FRONTEND=noninteractive "${APT[@]}" install -y --no-install-recommends \
  cage foot fonts-jetbrains-mono \
  network-manager wpasupplicant alsa-utils mpv \
  curl jq ca-certificates zram-tools \
  avahi-daemon libnss-mdns \
  python3 python3-regex stockfish fonts-dejavu-core gtypist \
  sudo
# Wifi firmware for common netbook chips (Atheros, Ralink, Realtek). Optional: skipped if unavailable.
DEBIAN_FRONTEND=noninteractive "${APT[@]}" install -y --no-install-recommends \
  firmware-misc-nonfree firmware-atheros firmware-realtek || echo "    (wifi firmware not available, skipped)"
[[ -n "$OFFLINE_LIST" ]] && rm -f "$OFFLINE_LIST"

# Asus Eee PC: load the driver that switches the wifi chip on, now and at every boot.
if grep -qi "asus" /sys/class/dmi/id/sys_vendor 2>/dev/null && grep -qi "eee" /sys/class/dmi/id/product_name 2>/dev/null; then
  modprobe eeepc_laptop 2>/dev/null || true
  grep -qx eeepc_laptop /etc/modules || echo eeepc_laptop >> /etc/modules
fi
nmcli radio wifi on 2>/dev/null || true

echo "==> Saving memory with zram"
sed -i 's/^#\?PERCENT=.*/PERCENT=50/' /etc/default/zramswap || true
systemctl enable --now zramswap.service || true

echo "==> Child account '$KIND' (no admin rights)"
if ! id "$KIND" &>/dev/null; then
  adduser --disabled-password --gecos "Unscroll" "$KIND"
fi
for g in audio video netdev render input; do getent group "$g" >/dev/null && usermod -aG "$g" "$KIND"; done
gpasswd -d "$KIND" sudo 2>/dev/null || true

echo "==> Placing files"
install -d -m 0755 /etc/unscroll /opt/unscroll/releases
install -m 0755 "$HIER/scripts/unscroll-sessie" /usr/local/bin/unscroll-sessie
install -m 0755 "$HIER/scripts/unscroll-update" /usr/local/sbin/unscroll-update
install -m 0755 "$HIER/scripts/unscroll-ouder" /usr/local/sbin/unscroll-ouder
[[ -f /etc/unscroll/ouders.yaml ]] || install -m 0644 "$HIER/config/ouders.example.yaml" /etc/unscroll/ouders.yaml
[[ -f /etc/unscroll/update.conf ]] || sed "s#^REPO=.*#REPO=\"$REPO\"#" "$HIER/config/update.example.conf" > /etc/unscroll/update.conf
[[ -f /etc/unscroll/geheimen.env ]] || { install -m 0640 -g "$KIND" "$HIER/config/geheimen.example.env" /etc/unscroll/geheimen.env; }
chown root:root /etc/unscroll/ouders.yaml /etc/unscroll/update.conf
chmod 0644 /etc/unscroll/ouders.yaml /etc/unscroll/update.conf

# both folders explicitly: install -d would otherwise create ~/.config as root,
# and then the app cannot save its settings
install -d -o "$KIND" -g "$KIND" "/home/$KIND/.config" "/home/$KIND/.config/foot"
chown "$KIND:$KIND" "/home/$KIND/.config"
install -m 0644 -o "$KIND" -g "$KIND" "$HIER/config/foot.ini" "/home/$KIND/.config/foot/foot.ini"

echo "==> Installing the app"
if [[ -x "$HIER/unscroll" ]]; then
  # binary shipped next to this script (for example built by hand)
  install -d /opt/unscroll/releases/lokaal
  install -m 0755 "$HIER/unscroll" /opt/unscroll/releases/lokaal/unscroll
  ln -sfn /opt/unscroll/releases/lokaal /opt/unscroll/huidig
else
  /usr/local/sbin/unscroll-update --nu || echo "No release found yet; the updater will try again later."
fi
ln -sfn /opt/unscroll/huidig/unscroll /usr/local/bin/unscroll
# Spotify speaker from the offline bundle (otherwise the updater brings it)
[[ -x "$HIER/librespot" ]] && install -m 0755 "$HIER/librespot" /usr/local/bin/librespot
# Hedy (programming) from the offline bundle (otherwise the updater brings it)
if [[ -f "$HIER/unscroll-hedy.tar.gz" ]]; then
  rm -rf /opt/unscroll/hedy && mkdir -p /opt/unscroll/hedy
  tar -xzf "$HIER/unscroll-hedy.tar.gz" --no-same-owner -C /opt/unscroll/hedy && chmod -R a+rX /opt/unscroll/hedy
fi

echo "==> Automatic login on tty1, starting the app right away"
install -d /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/kinderkamer.conf <<EOF
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin $KIND --noclear %I \$TERM
EOF
cat > "/home/$KIND/.bash_profile" <<'EOF'
# Start Unscroll on the screen (tty1 only).
if [[ "$(tty)" == "/dev/tty1" ]]; then
  exec /usr/local/bin/unscroll-sessie
fi
EOF
chown "$KIND:$KIND" "/home/$KIND/.bash_profile"

echo "==> Updater every 15 minutes"
install -m 0644 "$HIER/systemd/unscroll-update.service" /etc/systemd/system/
install -m 0644 "$HIER/systemd/unscroll-update.timer" /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now unscroll-update.timer

echo "==> Limiting and saving the volume"
amixer -q set Master 40% unmute 2>/dev/null || true
alsactl store 2>/dev/null || true

cat <<EOF

Done. Now set the PIN, quiet time and secrets locally with the parent helper:

  sudo unscroll-ouder

Passwords and keys only live in /etc/unscroll on this device, never in git.
Set up wifi with: sudo nmtui
Then reboot: sudo reboot
EOF
