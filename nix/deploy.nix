{ ... }:
{
  perSystem =
    {
      pkgs,
      self',
      ...
    }:
    {
      apps.deploy.program = pkgs.writeShellApplication {
        name = "deploy";
        runtimeInputs = [
          pkgs.openssh
          pkgs.docker
        ];
        text = ''
          set -euo pipefail

          IMAGE="ghcr.io/semi710/naste-server"
          CONTAINER="naste-server"
          DATA_DIR="/var/lib/naste-server/data"
          HOST="''${1:-}"

          deploy_cmds() {
            cat << 'REMOTE'
            set -euo pipefail
            
            IMAGE="__IMAGE__"
            CONTAINER="__CONTAINER__"
            DATA_DIR="__DATA_DIR__"
            
            if [ -n "__TOKEN__" ]; then
              echo "[deploy] Logging in to ghcr.io..."
              echo "__TOKEN__" | docker login ghcr.io -u github --password-stdin
            fi
            
            echo "[deploy] Pulling latest image..."
            docker pull "$IMAGE:latest"
            mkdir -p "$DATA_DIR"
            
            # Load config: env file > existing container > defaults
            PORT="8080"
            PRIVATE_USER=""
            PRIVATE_PASS=""
            
            if [ -f /etc/naste-server/env ]; then
              echo "[deploy] Loading config from /etc/naste-server/env..."
              while IFS='=' read -r key val; do
                [ -z "$key" ] && continue
                case "$key" in
                  PORT) PORT="$val" ;;
                  PRIVATE_USER) PRIVATE_USER="$val" ;;
                  PRIVATE_PASS) PRIVATE_PASS="$val" ;;
                esac
              done < /etc/naste-server/env
            fi
            
            if docker ps -a --format '{{.Names}}' | grep -q "^$CONTAINER$"; then
              echo "[deploy] Preserving existing container config..."
              C_PORT=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$CONTAINER" | grep '^PORT=' | cut -d= -f2 || echo "")
              C_USER=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$CONTAINER" | grep '^PRIVATE_USER=' | cut -d= -f2 || echo "")
              C_PASS=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$CONTAINER" | grep '^PRIVATE_PASS=' | cut -d= -f2 || echo "")
              [ -n "$C_PORT" ] && PORT="$C_PORT"
              [ -n "$C_USER" ] && PRIVATE_USER="$C_USER"
              [ -n "$C_PASS" ] && PRIVATE_PASS="$C_PASS"
              docker stop "$CONTAINER" 2>/dev/null || true
              docker rm "$CONTAINER" 2>/dev/null || true
            fi
            
            # Prompt only for unset values
            if [ -z "$PRIVATE_USER" ]; then
              read -rp "[deploy] PRIVATE_USER (leave empty for public-only): " PRIVATE_USER
            else
              echo "[deploy] PRIVATE_USER: ***"
            fi
            if [ -z "$PRIVATE_PASS" ]; then
              read -rsp "[deploy] PRIVATE_PASS (leave empty for public-only): " PRIVATE_PASS
              echo
            else
              echo "[deploy] PRIVATE_PASS: ***"
            fi
            read -rp "[deploy] PORT [$PORT]: " PORT_INPUT
            PORT=''${PORT_INPUT:-$PORT}
            
            # Persist config for next deploy
            mkdir -p /etc/naste-server
            {
              echo "PORT=$PORT"
              echo "PRIVATE_USER=$PRIVATE_USER"
              echo "PRIVATE_PASS=$PRIVATE_PASS"
            } > /etc/naste-server/env
            chmod 600 /etc/naste-server/env
            
            ENV_ARGS=(-e "PORT=$PORT")
            [ -n "$PRIVATE_USER" ] && ENV_ARGS+=(-e "PRIVATE_USER=$PRIVATE_USER")
            [ -n "$PRIVATE_PASS" ] && ENV_ARGS+=(-e "PRIVATE_PASS=$PRIVATE_PASS")
            
            docker run -d \
              --name "$CONTAINER" \
              --read-only \
              --cap-drop ALL \
              --security-opt no-new-privileges:true \
              -p "''${PORT}:8080" \
              "''${ENV_ARGS[@]}" \
              -v "''${DATA_DIR}:/data/paste" \
              --restart unless-stopped \
              "$IMAGE:latest"
            
            echo "[deploy] Done"
            echo "[deploy] Container: $CONTAINER"
            echo "[deploy] Port: $PORT"
            echo "[deploy] Data: $DATA_DIR"
            echo "[deploy] Config: /etc/naste-server/env"
          REMOTE
          }

          REMOTE_SCRIPT=$(deploy_cmds | sed \
            -e "s#__TOKEN__#''${GITHUB_TOKEN:-}#g" \
            -e "s#__IMAGE__#''${IMAGE}#g" \
            -e "s#__CONTAINER__#''${CONTAINER}#g" \
            -e "s#__DATA_DIR__#''${DATA_DIR}#g")

          if [ -z "$HOST" ]; then
            echo "[deploy] Updating locally..."
            eval "$REMOTE_SCRIPT"
          else
            if [ -t 0 ]; then
              echo "[deploy] Connecting to $HOST (interactive)..."
              ssh -t "$HOST" "bash --norc -s" <<< "$REMOTE_SCRIPT"
            else
              echo "[deploy] Connecting to $HOST (non-interactive)..."
              ssh -T "$HOST" "bash --norc -s" <<< "$REMOTE_SCRIPT"
            fi
          fi
        '';
      };
    };
}
