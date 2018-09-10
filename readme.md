### extract_vpns.go

`extract_vpns.go` will print VPN servers ip:port pairs from .ovpn files in a zip archive


```
go run extract_vpns.go -inzip="ovns.zip" > vpn_list.csv
```

To overwrite the outputed port to "1080"
```
go run extract_vpns.go -inzip="ovns.zip" -port="1080" > vpn_list.csv
```


### proxy_test_script.go

`proxy_test_script.go` connects to a list of socks5 proxy servers and attempts a http request to `https://httpbin.org/ip` or a configured url


```
go run proxy_test_script.go -config=secret\config.json vpn_list.csv
```

See `config.sample.json` for options.