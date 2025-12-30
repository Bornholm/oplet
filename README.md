# Oplet

Built by Devs, Run by Everyone.

## Getting started

### With Docker

```shell
docker volume create oplet_data
docker run --rm -it -p 3002:3002 -v /var/run/docker.sock:/var/run/docker.sock -v "oplet_data:/data" --env-file .env ghcr.io/bornholm/oplet:latest
```

With the following `.env` file:

```shell
OPLET_HTTP_BASE_URL=http://<public_host>:3002
OPLET_HTTP_SESSION_KEYS=<32_bytes_session_signing_key>

# OpenID Identity Provider configuration (ex: Google)
OPLET_HTTP_AUTHN_PROVIDERS_GOOGLE_KEY=<google_openid_key>
OPLET_HTTP_AUTHN_PROVIDERS_GOOGLE_SECRET=<google_openid_secret>

# Default administrator email
OPLET_HTTP_AUTHN_DEFAULT_ADMIN_EMAIL=<your_email>
```

## Documentation

### Tutorials

- [Creating an Oplet task](./doc/tutorials/creating-an-oplet-task.md)
