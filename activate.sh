#!/usr/bin/env sh
case ":$PATH:" in
  *":$(pwd):"*) ;;
  *) PATH="$(pwd):$PATH"; export PATH ;;
esac
echo "Stockbridge is active for this shell session. Try: stockbridge help"
