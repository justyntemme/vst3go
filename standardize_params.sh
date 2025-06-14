#!/bin/bash

# Standardize parameter constant declarations across all plugins

echo "Standardizing parameter constants..."

# Fix chain_fx
echo "Updating chain_fx..."
sed -i 's/const (/const (\n\t\/\/ Parameter IDs/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramBypass = iota/ParamBypass uint32 = iota/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramGateThreshold/ParamGateThreshold/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramCompThreshold/ParamCompThreshold/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramCompRatio/ParamCompRatio/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramNoiseAmount/ParamNoiseAmount/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go
sed -i 's/paramChainSelect/ParamChainSelect/' /home/user/Documents/code/vst3go/examples/chain_fx/main.go

# Fix debug_example
echo "Updating debug_example..."
sed -i 's/const (/const (\n\t\/\/ Parameter IDs/' /home/user/Documents/code/vst3go/examples/debug_example/main.go
sed -i 's/paramBypass = iota/ParamBypass uint32 = iota/' /home/user/Documents/code/vst3go/examples/debug_example/main.go
sed -i 's/paramDebugLevel/ParamDebugLevel/' /home/user/Documents/code/vst3go/examples/debug_example/main.go

# Fix gain plugin missing uint32
echo "Updating gain..."
sed -i 's/ParamGain = iota/ParamGain uint32 = iota/' /home/user/Documents/code/vst3go/examples/gain/main.go

echo "Done standardizing parameter constants!"