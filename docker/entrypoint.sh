#!/usr/bin/env bash

DATA_DIR="${DATA_DIR:-/data}"
SUBNETS="/app/cf.local.iplist"

mkdir -p "$DATA_DIR"

SCAN_PID=""
CANCEL_REQUESTED=0

screen_clear() {
    printf '\033[2J\033[H'
}

pause_menu() {
    echo
    read -r -p " Press Enter to continue..." _
}

handle_interrupt() {
    if [ -n "${SCAN_PID:-}" ] && kill -0 "$SCAN_PID" 2>/dev/null; then
        CANCEL_REQUESTED=1
        echo
        echo " Cancelling current scan..."
        kill -TERM "$SCAN_PID" 2>/dev/null || true
        return
    fi

    echo
    exit 130
}

trap handle_interrupt INT TERM

render_header() {
    screen_clear

    echo "╔════════════════════════════════════════════════════╗"
    printf "║ %-50s ║\n" "CFScanner Universal v1.0.0"
    printf "║ %-50s ║\n" ""
    printf "║ %-50s ║\n" "VLESS / VMess / Trojan / Cloudflare"
    printf "║ %-50s ║\n" ""
    printf "║ %-50s ║\n" "Learned from: Morteza Bashsiz / CFScanner"
    printf "║ %-50s ║\n" "Extended by : Ahmad Sheikhi"
    printf "║ %-50s ║\n" "Senior Platform / DevOps Engineer"
    echo "╚════════════════════════════════════════════════════╝"
}

reset_scan_state() {
    rm -rf \
        "$DATA_DIR/work" \
        "$DATA_DIR/seen-ips.txt" \
        "$DATA_DIR/working-raw.txt" \
        "$DATA_DIR/working-ranked.txt" \
        "$DATA_DIR"/top*.txt \
        "$DATA_DIR"/top*-ranked.txt \
        "$DATA_DIR"/top*-links.txt \
        "$DATA_DIR/proxy-url.txt" \
        "$DATA_DIR/config.json" \
        "$DATA_DIR/config-summary.txt"
}

parse_proxy_config() {
    echo
    read -r -p " Paste VLESS / VMess / Trojan config: " proxy_url

    case "$proxy_url" in
        vless://*|vmess://*|trojan://*)
            ;;
        *)
            echo
            echo " ERROR: Unsupported config."
            echo " Supported: VLESS / VMess / Trojan"
            return 1
            ;;
    esac

    old_url=""
    if [ -f "$DATA_DIR/proxy-url.txt" ]; then
        old_url="$(cat "$DATA_DIR/proxy-url.txt")"
    fi

    if [ -n "$old_url" ] && [ "$old_url" != "$proxy_url" ]; then
        echo
        echo " New proxy config detected."
        echo " Resetting previous scan state..."
        reset_scan_state
    fi

    printf '%s\n' "$proxy_url" > "$DATA_DIR/proxy-url.txt"

    python3 - "$proxy_url" "$DATA_DIR/config.json" "$DATA_DIR/config-summary.txt" <<'PY'
import base64
import json
import sys
from urllib.parse import urlparse, parse_qs, unquote

raw = sys.argv[1].strip()
config_path = sys.argv[2]
summary_path = sys.argv[3]

def b64decode_text(value):
    value = value.strip()
    value += "=" * (-len(value) % 4)
    try:
        return base64.urlsafe_b64decode(value).decode()
    except Exception:
        return base64.b64decode(value).decode()

protocol = ""
address = ""
port = "443"
user_id = ""
network = "ws"
host = ""
path = ""
sni = ""
security = "none"

if raw.startswith("vmess://"):
    protocol = "vmess"

    payload = raw[len("vmess://"):]
    obj = json.loads(b64decode_text(payload))

    address = str(obj.get("add", ""))
    port = str(obj.get("port", "443"))
    user_id = str(obj.get("id", ""))

    network = str(obj.get("net", "ws") or "ws").lower()
    host = str(obj.get("host", "") or "")
    path = str(obj.get("path", "") or "")
    sni = str(obj.get("sni", "") or host)

    tls_value = str(obj.get("tls", "") or "").lower()
    security = "tls" if tls_value not in ("", "none", "false", "0") else "none"

else:
    u = urlparse(raw)
    q = parse_qs(u.query)

    protocol = u.scheme.lower()
    address = u.hostname or ""
    port = str(u.port or 443)
    user_id = unquote(u.username or "")

    network = q.get("type", ["ws"])[0].lower()
    host = q.get("host", [""])[0]
    path = unquote(q.get("path", [""])[0])
    sni = q.get("sni", [""])[0]

    if protocol == "trojan":
        security = q.get("security", ["tls"])[0].lower()
    else:
        security = q.get("security", ["none"])[0].lower()

    if not sni:
        sni = host

if network not in ("ws", "grpc", "xhttp"):
    raise SystemExit(f"Unsupported transport: {network}")

if security not in ("tls", "none", ""):
    raise SystemExit(f"Unsupported security for this scanner: {security}")

if not user_id:
    raise SystemExit("Missing UUID/password")

if not host:
    host = sni

if not sni:
    sni = host

tls_enabled = security == "tls"

cfg = {
    "localPort": 10808,
    "addressPort": port,
    "userId": user_id,
    "protocol": protocol,
    "network": network,
    "tls": tls_enabled,
    "security": security or "none",
    "xhttpMode": "auto",
    "randomizeSni": False,
    "wsHeaderHost": host,
    "wsHeaderPath": path,
    "sni": sni,
    "frontingTimeout": 5.0,
    "nTries": 1,
    "writer": "csv"
}

with open(config_path, "w") as f:
    json.dump(cfg, f, indent=2)

summary = [
    f"Protocol  : {protocol.upper()}",
    f"Address   : {address}",
    f"Port      : {port}",
    f"Transport : {network}",
    f"TLS       : {'enabled' if tls_enabled else 'disabled'}",
    f"Host      : {host}",
    f"SNI       : {sni}",
    f"Path      : {path}",
]

with open(summary_path, "w") as f:
    f.write("\n".join(summary) + "\n")

print()
print("==============================================")
print(" CONFIG ACCEPTED")
print("==============================================")
for line in summary:
    print(" " + line)
print("==============================================")
print(f" Scanner config: {config_path}")
PY
}

generate_candidate_batch() {
    local output="$1"
    local batch_size="$2"

    python3 - \
        "$SUBNETS" \
        "$DATA_DIR/seen-ips.txt" \
        "$output" \
        "$batch_size" <<'PY'
import ipaddress
import random
import sys

src, seen_path, output_path, count = sys.argv[1:]
count = int(count)

nets = []

with open(src) as f:
    for line in f:
        line = line.strip()

        if not line or line.startswith("#"):
            continue

        net = ipaddress.ip_network(line, strict=False)

        if net.version == 4:
            nets.append(net)

seen = set()

try:
    with open(seen_path) as f:
        seen = {x.strip() for x in f if x.strip()}
except FileNotFoundError:
    pass

batch = set()

while len(batch) < count:
    net = random.choice(nets)
    ip = str(net[random.randrange(net.num_addresses)])

    if ip in seen or ip in batch:
        continue

    batch.add(ip)

with open(output_path, "w") as f:
    for ip in sorted(batch):
        f.write(ip + "\n")

with open(seen_path, "a") as f:
    for ip in sorted(batch):
        f.write(ip + "\n")
PY
}

normalize_ranked_results() {
    local raw="$DATA_DIR/working-raw.txt"
    local ranked="$DATA_DIR/working-ranked.txt"

    [ -f "$raw" ] || return 0

    awk '
        NF >= 2 {
            latency = $1 + 0
            ip = $2

            if (!(ip in best) || latency < best[ip]) {
                best[ip] = latency
            }
        }

        END {
            for (ip in best) {
                print best[ip], ip
            }
        }
    ' "$raw" |
    sort -n > "$ranked"
}

working_count() {
    if [ -s "$DATA_DIR/working-ranked.txt" ]; then
        wc -l < "$DATA_DIR/working-ranked.txt"
    else
        echo 0
    fi
}

render_scan_dashboard() {
    local mode="$1"
    local target="$2"
    local round="$3"
    local scanned="$4"
    local working="$5"
    local batch="$6"

    screen_clear

    echo "╔════════════════════════════════════════════════════╗"
    echo "║              CFScanner Universal v1.0.0                  ║"
    echo "╠════════════════════════════════════════════════════╣"
    printf "║  Mode        : %-35s ║\n" "$mode"
    printf "║  Target      : %-35s ║\n" "$target working IPs"
    printf "║  Round       : %-35s ║\n" "$round"
    printf "║  Batch       : %-35s ║\n" "$batch IPs"
    printf "║  Scanned     : %-35s ║\n" "$scanned"
    printf "║  Working     : %-35s ║\n" "$working / $target"
    echo "╠════════════════════════════════════════════════════╣"
    echo "║  Current Best IPs - Top 10                         ║"
    echo "╚════════════════════════════════════════════════════╝"

    if [ -s "$DATA_DIR/working-ranked.txt" ]; then
        head -10 "$DATA_DIR/working-ranked.txt" |
        awk '{
            printf " %2d. %-16s %6sms\n", NR, $2, $1
        }'
    else
        echo
        echo " No working IP found yet."
    fi

    echo
}

run_scanner_batch() {
    local batch_file="$1"
    local threads="$2"
    local log_file="$3"

    CANCEL_REQUESTED=0

    (
        cd "$DATA_DIR/work" || exit 1

        exec env \
            TERM=dumb \
            NO_COLOR=1 \
            cfscanner \
              --config "$DATA_DIR/config.json" \
              --subnets "$batch_file" \
              --threads "$threads" \
              --tries 1 \
              --vpn
    ) >"$log_file" 2>&1 &

    SCAN_PID=$!

    local frames='|/-\'
    local i=0

    while kill -0 "$SCAN_PID" 2>/dev/null; do
        printf "\r Scanning %c  Ctrl+C = cancel " \
            "${frames:i%4:1}"

        i=$((i + 1))
        sleep 0.5
    done

    wait "$SCAN_PID" 2>/dev/null || true
    SCAN_PID=""

    printf "\r%-55s\r" " "

    if [ "$CANCEL_REQUESTED" -eq 1 ]; then
        echo
        echo " Scan cancelled."
        return 130
    fi

    return 0
}

scan_target() {
    local target="$1"
    local base_batch="$2"
    local threads="$3"
    local mode="$4"

    if [ ! -f "$DATA_DIR/config.json" ]; then
        echo
        echo " ERROR: No proxy config."
        echo " Use [1] Paste Proxy Config first."
        pause_menu
        return
    fi

    if [ ! -f "$SUBNETS" ]; then
        echo
        echo " ERROR: Cloudflare range list is missing."
        pause_menu
        return
    fi

    mkdir -p "$DATA_DIR/work"
    touch \
        "$DATA_DIR/seen-ips.txt" \
        "$DATA_DIR/working-raw.txt"

    normalize_ranked_results

    local round=0
    local current
    local scanned
    local remaining
    local this_batch

    current="$(working_count)"

    while [ "$current" -lt "$target" ]; do
        round=$((round + 1))
        remaining=$((target - current))

        this_batch="$base_batch"

        # Near target, reduce the amount of unnecessary scanning.
        if [ "$remaining" -le 3 ]; then
            this_batch=10
        elif [ "$remaining" -le 7 ]; then
            this_batch=20
        elif [ "$remaining" -le 12 ] && [ "$this_batch" -gt 30 ]; then
            this_batch=30
        fi

        generate_candidate_batch \
            "$DATA_DIR/work/batch.txt" \
            "$this_batch"

        scanned="$(wc -l < "$DATA_DIR/seen-ips.txt")"

        render_scan_dashboard \
            "$mode" \
            "$target" \
            "$round" \
            "$scanned" \
            "$current" \
            "$this_batch"

        LOG="$DATA_DIR/work/round-${round}.log"

        run_scanner_batch \
            "$DATA_DIR/work/batch.txt" \
            "$threads" \
            "$LOG"

        rc=$?

        if [ "$rc" -eq 130 ]; then
            pause_menu
            return
        fi

        FINAL="$(
            ls -1t \
                "$DATA_DIR/work/result/"*_final.txt \
                2>/dev/null |
            head -1
        )"

        if [ -z "$FINAL" ] || [ ! -f "$FINAL" ]; then
            echo
            echo " ERROR: Scanner did not produce a final result."
            echo
            echo " Last scanner log:"
            tail -20 "$LOG"
            pause_menu
            return
        fi

        cat "$FINAL" >> "$DATA_DIR/working-raw.txt"

        normalize_ranked_results

        current="$(working_count)"
        scanned="$(wc -l < "$DATA_DIR/seen-ips.txt")"

        render_scan_dashboard \
            "$mode" \
            "$target" \
            "$round" \
            "$scanned" \
            "$current" \
            "$this_batch"

        if [ "$current" -lt "$target" ]; then
            echo " Target not reached yet. Starting next batch..."
            sleep 2
        fi
    done

    ranked_out="$DATA_DIR/top${target}-ranked.txt"
    ip_out="$DATA_DIR/top${target}.txt"

    head -n "$target" \
        "$DATA_DIR/working-ranked.txt" \
        > "$ranked_out"

    awk '{print $2}' \
        "$ranked_out" \
        > "$ip_out"

    screen_clear

    echo "╔════════════════════════════════════════════════════╗"
    echo "║                  SCAN COMPLETE                     ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo
    echo " Mode       : $mode"
    echo " Target     : $target"
    echo " Scanned    : $(wc -l < "$DATA_DIR/seen-ips.txt")"
    echo " Working    : $(working_count)"
    echo
    echo " Top $target:"
    echo

    awk '{
        printf " %3d. %-16s %6sms\n", NR, $2, $1
    }' "$ranked_out"

    echo
    echo " Files:"
    echo "   $ip_out"
    echo "   $ranked_out"
}

latest_ranked_file() {
    ls -1t "$DATA_DIR"/top*-ranked.txt \
        2>/dev/null |
    head -1
}

export_client_links() {
    local ranked
    ranked="$(latest_ranked_file)"

    if [ -z "$ranked" ] || [ ! -f "$ranked" ]; then
        echo
        echo " ERROR: No Top results available."
        return 1
    fi

    if [ ! -f "$DATA_DIR/proxy-url.txt" ]; then
        echo
        echo " ERROR: Original proxy config is missing."
        return 1
    fi

    output="${ranked%-ranked.txt}-links.txt"

    python3 - \
        "$DATA_DIR/proxy-url.txt" \
        "$ranked" \
        "$output" <<'PY'
import base64
import json
import sys
from urllib.parse import (
    urlparse,
    urlunparse,
    quote,
)

proxy_file, ranked_file, output_file = sys.argv[1:]

raw = open(proxy_file).read().strip()

rows = []

with open(ranked_file) as f:
    for line in f:
        parts = line.split()

        if len(parts) >= 2:
            rows.append((parts[0], parts[1]))

def decode_vmess(value):
    value += "=" * (-len(value) % 4)
    return json.loads(
        base64.urlsafe_b64decode(value).decode()
    )

links = []

if raw.startswith("vmess://"):
    original = decode_vmess(raw[len("vmess://"):])

    for idx, (latency, ip) in enumerate(rows, 1):
        item = dict(original)
        item["add"] = ip
        item["ps"] = f"🇬🇧 CF {idx:03d} | {latency}ms"

        encoded = base64.b64encode(
            json.dumps(
                item,
                ensure_ascii=False,
                separators=(",", ":")
            ).encode()
        ).decode()

        links.append("vmess://" + encoded)

else:
    u = urlparse(raw)

    for idx, (latency, ip) in enumerate(rows, 1):
        name = f"🇬🇧 CF {idx:03d} | {latency}ms"

        username = quote(
            u.username or "",
            safe=""
        )

        port = u.port or 443

        netloc = f"{username}@{ip}:{port}"

        link = urlunparse((
            u.scheme,
            netloc,
            u.path,
            "",
            u.query,
            quote(name, safe="")
        ))

        links.append(link)

with open(output_file, "w") as f:
    f.write("\n".join(links) + "\n")

print(f" Exported {len(links)} client links")
print(f" Output: {output_file}")
PY
}

show_top_results() {
    local ranked
    ranked="$(latest_ranked_file)"

    if [ -z "$ranked" ] || [ ! -f "$ranked" ]; then
        echo
        echo " No Top results yet."
        return
    fi

    echo
    echo " Latest Top Results"
    echo

    head -100 "$ranked" |
    awk '{
        printf " %3d. %-16s %6sms\n", NR, $2, $1
    }'
}

custom_ip_scan() {
    if [ ! -f "$DATA_DIR/config.json" ]; then
        echo
        echo " ERROR: Use option [1] first."
        return
    fi

    echo
    read -r -p \
        " File inside /data [custom-ips.txt]: " \
        custom_name

    custom_name="${custom_name:-custom-ips.txt}"
    custom_file="$DATA_DIR/$custom_name"

    if [ ! -f "$custom_file" ]; then
        echo
        echo " ERROR: $custom_file not found."
        return
    fi

    read -r -p " Threads [10]: " custom_threads
    custom_threads="${custom_threads:-10}"

    mkdir -p "$DATA_DIR/work"

    log="$DATA_DIR/work/custom.log"

    render_header

    echo
    echo " Custom IP Scan"
    echo " Source: $custom_file"
    echo

    run_scanner_batch \
        "$custom_file" \
        "$custom_threads" \
        "$log"

    rc=$?

    if [ "$rc" -eq 130 ]; then
        return
    fi

    final="$(
        sed -n \
            's/^Sorted IPS Written in : //p' \
            "$log" |
        tail -1
    )"

    if [ -n "$final" ] && [ -f "$final" ]; then
        cp "$final" "$DATA_DIR/custom-ranked.txt"
        awk '{print $2}' \
            "$final" \
            > "$DATA_DIR/custom.txt"

        echo
        echo " Results:"
        head -20 "$final"
    else
        echo
        echo " No final result produced."
        tail -20 "$log"
    fi
}

while true; do
    render_header

    if [ -f "$DATA_DIR/config-summary.txt" ]; then
        echo
        echo " Current Config"
        echo " ──────────────────────────────────────────────"
        sed 's/^/  /' "$DATA_DIR/config-summary.txt"
        echo " ──────────────────────────────────────────────"
    fi

    echo
    range_count="$(grep -vcE '^[[:space:]]*(#|$)' "$SUBNETS" 2>/dev/null || echo 0)"
    echo " Cloudflare IPv4 Ranges: $range_count CIDRs"
    echo
    echo " [1] Paste Proxy Config"
    echo " [2] Quick Scan       - Find 20 Working IPs"
    echo " [3] Standard Scan    - Find 50 Working IPs"
    echo " [4] Deep Scan        - Find 100 Working IPs"
    echo " [5] Custom Scan"
    echo " [6] Show Top Results"
    echo " [7] Export Client Links"
    echo " [8] Scan Custom IP List"
    echo " [9] Reset All Data"
    echo " [10] Show Cloudflare Ranges"
    echo " [0] Exit"
    echo

    read -r -p " Select: " choice

    case "$choice" in
        1)
            parse_proxy_config
            pause_menu
            ;;

        2)
            scan_target 20 50 10 "Quick Scan"
            pause_menu
            ;;

        3)
            scan_target 50 100 15 "Standard Scan"
            pause_menu
            ;;

        4)
            scan_target 100 200 20 "Deep Scan"
            pause_menu
            ;;

        5)
            echo
            read -r -p " Target working IPs [100]: " target
            target="${target:-100}"

            read -r -p " Batch size [100]: " batch
            batch="${batch:-100}"

            read -r -p " Threads [20]: " threads
            threads="${threads:-20}"

            scan_target \
                "$target" \
                "$batch" \
                "$threads" \
                "Custom Scan"

            pause_menu
            ;;

        6)
            show_top_results
            pause_menu
            ;;

        7)
            export_client_links

            latest_links="$(
                ls -1t "$DATA_DIR"/top*-links.txt \
                    2>/dev/null |
                head -1
            )"

            if [ -n "$latest_links" ]; then
                echo
                echo " Preview: first 10 exported links:"
                echo
                head -10 "$latest_links"
            fi

            pause_menu
            ;;

        8)
            custom_ip_scan
            pause_menu
            ;;

        9)
            echo
            read -r -p \
                " Clear proxy config and all scan data? [y/N]: " \
                confirm

            case "$confirm" in
                y|Y|yes|YES)
                    reset_scan_state
                    echo
                    echo " All scan data and proxy config cleared."
                    ;;
                *)
                    echo
                    echo " Cancelled."
                    ;;
            esac

            pause_menu
            ;;

        10)
            render_header

            echo
            echo " Cloudflare IPv4 Ranges"
            echo " ──────────────────────────────────────────────"

            nl -w2 -s'. ' "$SUBNETS"

            echo " ──────────────────────────────────────────────"
            echo " Total: $(grep -vcE '^[[:space:]]*(#|$)' "$SUBNETS") CIDRs"

            pause_menu
            ;;

        0)
            echo
            echo " Bye."
            exit 0
            ;;

        *)
            echo
            echo " Invalid selection."
            pause_menu
            ;;
    esac
done
