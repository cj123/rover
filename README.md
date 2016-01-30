rover
=====

a cmd line util for [ranger](http://github.com/DHowett/ranger)


```
Usage of ./rover:
  -b uint
    	limit filesize downloaded (in bytes)
  -l	list files in zip
  -o string
    	the output filename
  -r string
    	the remote filename to download
  -t int
    	timeout, in seconds (default 5)
  -u string
    	the url you wish to download from
  -v	verbose
```

e.g.

```shell
./rover -u `curl https://api.ipsw.me/v2.1/iPhone5,1/latest/url` -r Restore.plist -o -
```
