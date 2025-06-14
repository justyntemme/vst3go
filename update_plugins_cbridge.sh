#!/bin/bash

# Update all example plugins to use the cbridge package instead of direct C imports

EXAMPLES_DIR="examples"
PLUGINS=(
    "delay"
    "filter"
    "simplesynth"
    "transientshaper"
    "studiogate"
    "mastercompressor"
    "debug_example"
    "chain_fx"
    "vocalstrip"
    "drumbus"
    "jetflanger"
    "vintagechorus"
    "masterlimiter"
    "auto_params"
    "surround"
    "sidechain"
    "multidistortion"
)

for plugin in "${PLUGINS[@]}"; do
    PLUGIN_DIR="$EXAMPLES_DIR/$plugin"
    MAIN_FILE="$PLUGIN_DIR/main.go"
    
    if [ -f "$MAIN_FILE" ]; then
        echo "Updating $plugin..."
        
        # Remove C imports and add cbridge import
        sed -i '
            # Remove C cgo directives and imports
            /^\/\/ #cgo CFLAGS:/,/^import "C"$/d
            
            # Add cbridge import after other imports
            /^import (/,/^)/ {
                /^)/ i\
\t\
\t// Import C bridge - required for VST3 plugin to work\
\t_ "github.com/justyntemme/vst3go/pkg/plugin/cbridge"
            }
        ' "$MAIN_FILE"
        
        echo "  ✓ Updated $plugin"
    else
        echo "  ✗ Skipping $plugin (main.go not found)"
    fi
done

echo "Done updating plugins!"