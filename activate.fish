set stockbridge_dir (dirname (status --current-filename))
set stockbridge_dir (realpath "$stockbridge_dir")

if not contains "$stockbridge_dir" $PATH
    set -gx PATH "$stockbridge_dir" $PATH
end

echo "Stockbridge is active for this fish session. Try: stockbridge help"
