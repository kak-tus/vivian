# vivian

Simple script to connect to VPN, ping some IP in VPN network, and reconnect if ping fail.

This is like bash script, but in Go.

```
vivian -conn Office -ping 192.168.1.10
```

Script requires Network Manager, nmcli and ping commands.

"ping-restart" script builded in Network Manager doesn't work fine in unstable network - it do reconnect only once.
