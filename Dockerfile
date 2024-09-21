FROM scratch

COPY ./xiond /usr/bin/xiond

WORKDIR /root/.xion

# rest server
EXPOSE 1317
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

ENTRYPOINT [ "/usr/bin/xiond" ]

CMD [ "help" ]
