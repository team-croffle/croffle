#!/usr/bin/env bash
# Lists what apps/desktop/DESIGN.md forbids in the renderer screens. Prints nothing when clean,
# except the one allowed line (the title bar close button's red hover).
# Usage: apps/desktop/scripts/check-ui-tokens.sh
cd "$(dirname "$0")/../src/renderer/src" || exit 2
palette='slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|white|black'
echo '# palette colours, hex values, rgba()'
grep -rnE "(bg|text|border|ring)-(${palette})(-[0-9]+)?(/[0-9]+)?\b|\[#[0-9A-Fa-f]{3,8}\]|rgba\(" \
  --include='*.vue' --include='*.ts' --exclude-dir=ui .
echo '# text-destructive-foreground (only over bg-destructive)'
grep -rn 'text-destructive-foreground' --include='*.vue' --include='*.ts' .
echo '# browser confirm()'
grep -rnE '\bconfirm\(' --include='*.vue' --include='*.ts' --exclude-dir=ui .
exit 0
