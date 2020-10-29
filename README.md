[![golangci-lint Actions Status](https://github.com/lindenlab/caddy-s3-proxy/workflows/golangci-lint/badge.svg)](https://github.com/lindenlab/caddy-s3-proxy/actions)
[![Test Actions Status](https://github.com/lindenlab/caddy-s3-proxy/workflows/Test/badge.svg)](https://github.com/lindenlab/caddy-s3-proxy/actions)
[![All Contributors](https://img.shields.io/badge/all_contributors-2-orange.svg?style=flat-square)](#contributors)

# caddy-s3-proxy

caddy-s3-proxy allows you to proxy requests directly from S3.

S3 does have the website option, in which case, a normal reverse proxy could be used to display S3 data.
However, it is sometimes inconvient to do that.  This module lets you access S3 data even if website access
is not configured on your bucket.

## Making a version of caddy with this plugin

With caddy 2 you can use [xcaddy](https://github.com/caddyserver/xcaddy) to build a version of caddy
with this plugin installed.  To install xcaddy do:
```
go get -u github.com/caddyserver/xcaddy/cmd/xcaddy
```

This repo has a Makefile to make it easier to build a new version of caddy with this plugin.  Just type:
```
make build
```

You can run ```make docker``` do build a local image you can test with.

## Configuration
The Caddyfile directive would look something like this:
```
	s3proxy [<matcher>] {
		bucket <bucket_name>
		region <region_name>
		index  <list of index file names>
		endpoint <alternative S3 endpoint>
		root   <key prefix>
		enable_put
		enable_delete
		error_page <http status> <S3 key to a custom error page for this http status>
		error_page <S3 key to a default error page>
		browse [<path to template>]
	}
```

|  option   |  type  |  required | default | help |
|-----------|:------:|-----------|---------|------|
| bucket              | string   | yes |                          | S3 bucket name |
| region              | string   | no  |  env AWS_REGION          | S3 region - if not give in the Caddyfile then AWS_REGION env var must be set.|
| endpoint            | string   | no  |  aws default             | S3 hostname |
| index               | string[] | no  |  [index.html, index.txt] | Index files to look up for dir path |
| root                | string   | no  |    | Set a "prefix" to be added to key |
| enable_put          | bool     | yes | false   | Allow PUT method to be sent through proxy |
| enable_delete       | bool     | yes | false   | Allow DELETE method to be sent through proxy |
| error_page          | [int, ] string | no |  | Custom error page |
| browse              | [string] | no |  | Turns on a directory view for partial keys, an optional path to a template can be given |

## Credentials

This module uses the default providor chain to get credentials for access to S3.  This provides several more
secure options to provide credentials for accessing S3 without putting the credentials in the Caddyfile.
The methods include (and are looked for in this order):

1) Environment variables.  I.e. AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY

2) Shared credentials file.  (Located at ~/.aws/credentials)

3) If your application uses an ECS task definition or RunTask API operation, IAM role for tasks.

4) If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

For much more detail on the various options for setting AWS credentials see here:
https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html

## Examples you can play with

In the examples directory is an example of using the s3proxy with localstack.
Localstack contains a working version of S3 you can use for local development.

Check out the examples [here](example/LOCALSTACK_EXAMPLE.md).
You can also just run ```make example``` to build a docker image with the plugin and launch the compose example.

# Contributors

A big thank you to folks who have contributed to this project!

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="https://github.com/rayjlinden"><img src="https://avatars0.githubusercontent.com/u/42587610?v=4" width="100px;" alt=""/><br /><sub><b>rayjlinden</b></sub></a></td>
    <td align="center"><a href="https://github.com/gilbsgilbs"><img src="https://avatars2.githubusercontent.com/u/3407667?v=4" width="100px;" alt=""/><br /><sub><b>Gilbert Gilb's</b></sub></a></td>
  </tr>
</table>

<!-- markdownlint-enable -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

