cd ~/ibc-testnets

echo "\n*** transfer...\n"
echo 12345678 | gaiacli --home ibc-gaia/n0/gaiacli tx ibcmockbank transfer \
    --src-port bank --src-channel chann-to-iris \
    --denom uatom --amount 1 \
    --receiver $(iriscli --home ibc-iris/n0/iriscli keys show n0 | jq -r '.address') \
    --source true \
    --from n0 -y -o json \
    --broadcast-mode=block \
    >ibc-gaia/n0/result.json
jq -r '.events[1].attributes[5].value' ibc-gaia/n0/result.json >ibc-iris/n0/packet.json
echo "\n*** receive...\n"
sleep 8 && gaiacli --home ibc-gaia/n0/gaiacli q ibc client header -o json >ibc-iris/n0/header.json
gaiacli --home ibc-gaia/n0/gaiacli q ibc channel packet-proof bank chann-to-iris \
    $(jq -r '.m_sequence' ibc-iris/n0/packet.json) \
    $(jq -r '.value.SignedHeader.header.height' ibc-iris/n0/header.json) \
    -o json >ibc-iris/n0/proof.json
echo 12345678 | iriscli --home ibc-iris/n0/iriscli tx ibc client update client-to-gaia ibc-iris/n0/header.json --from n0 -y --broadcast-mode=block
echo 12345678 | iriscli --home ibc-iris/n0/iriscli tx ibcmockbank recv-packet \
    ibc-iris/n0/packet.json ibc-iris/n0/proof.json \
    $(jq -r '.value.SignedHeader.header.height' ibc-iris/n0/header.json) \
    --from n0 -y \
    --broadcast-mode=block
echo "\n*** query account...\n"
gaiacli --home ibc-gaia/n0/gaiacli q account -o text $(gaiacli --home ibc-gaia/n0/gaiacli keys show n0 | jq -r '.address')
iriscli --home ibc-iris/n0/iriscli q account -o text $(iriscli --home ibc-iris/n0/iriscli keys show n0 | jq -r '.address')
