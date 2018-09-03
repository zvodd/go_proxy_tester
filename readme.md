### extract_vpns.go

`extract_vpns.go` will print VPN servers ip:port pairs from .ovpn files in a zip archive - currently hard coded to `list.zip`


```
go run extract_vpns.go > vpn_list.csv

```


### proxy_test_script.go

`proxy_test_script.go` connects to a list of socks5 proxy servers and attempts a http request to `https://httpbin.org/ip` or a configured url


```
go run proxy_test_script.go -config=secret\config.json vpn_list.csv
```