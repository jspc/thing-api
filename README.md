# Thing API

The Thing API provides a simple API for writing developer coding interviews against.

The API is described in `docs/` in swagger format.

## Running

This API is packaged and distributed as a docker image:

```bash
$ docker pull jspc/thing-api
```

It may be run as per:

```bash
$ docker run -p 8080:8080 jspc/thing-api
```

## Building

The API can be built using the standard go toolchain, wrapped by Make:

```bash
$ make
```
