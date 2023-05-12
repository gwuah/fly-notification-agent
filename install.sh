
set -e

uri="https://github.com/gwuah/fly-notification-agent/releases/download/v0.0.1/fly-notification-agent_0.0.1_linux_amd64.tar.gz"
bin_dir="$HOME/.fly"

exe="$bin_dir/fly-notification-agent"

curl -q --fail --location --progress-bar --output "$exe.tar.gz" "$uri"
cd "$bin_dir"
tar xzf "$exe.tar.gz"
chmod +x "$exe"
rm "$exe.tar.gz"

echo "fly-notification-agent was installed successfully to $exe"
