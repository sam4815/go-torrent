#  go-torrent



https://github.com/sam4815/go-torrent/assets/32017929/be3315f4-7ffc-476d-9f34-668b048dbe62



## About

This project is a partial implementation of the BitTorrent protocol. Given a torrent file, `go-torrent` is able to find peers and download the associated files piece by piece. This was intended as a way to more deeply understand a protocol I've always been curious about.

The client supports HTTP and UDP trackers, multi-file torrents, and the ability to pause and resume downloads. At the moment, it only leeches - that is, it doesn't support uploading pieces - but this is something I would like to revisit at some point in the future.

## Usage

To initiate a download:

```
go run main.go --file ./path/to/my/torrent
```

You can also pass a `debug` flag to see the requests being made under the hood.

## Resources
1. https://blog.jse.li/posts/torrent/
1. https://wiki.theory.org/BitTorrentSpecification
