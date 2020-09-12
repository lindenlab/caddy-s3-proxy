# caddy-s3-proxy

caddy-s3-proxy allows you to proxy requests directly from S3.

S3 does have the website option, in which case, a normal reverse proxy could be used to display S3 data.
However, it is sometimes inconvient to do that.  This module lets you access S3 data even if website access
is not configured on your bucket.

## Making a version of caddy with this plugin

With caddy 2 you can use [xcaddy](https://github.com/caddyserver/xcaddy) to build a version of caddy
with this plugin installed.  The syntax would look something like this:
```
xcaddy build \
        --output /usr/local/bin/caddy \
        --with github.com/lindenlab/caddy-s3-proxy 
```

## Configuration
The Caddyfile directive would look something like this:
```
	s3proxy {
		bucket <bucket_name>
		region <region_name>
		index  <list of index file names>
                endpoint <alternative S3 endpoint>
	}
```

|  option   |  type  |  required | default | help |
|-----------|:------:|-----------|---------|------|
| bucket              | string   | yes |                          | S3 bucket |
| endpoint            | string   | no  |  aws default             | S3 hostname |
| region              | string   | no  |  env AWS_REGION          | S3 region |
| index               | string[] | no  |  [index.html, index.txt] | Index files to look up for dir path |

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

## Manipulating the path and resulting S3 key

In general, the path passed in to the module is used as the key to get an object from a bucket.  However,
you may want to serve the S3 data from some other directory in yur web site.  You can do that with the
uri directive.  For example:
```
        route /test-results/* {
                uri strip_prefix /results
                s3proxy {
                        region "us-west-2"
                        bucket "test-results.tilia-inc.com"
                }
        }
```
In this example a web request of *http://www.nysite.com/results/myresults.csv* would request a key from the S3 bucket of */myresults.csv*.
Whereas, if the uri directive was not present it would would request */results/myresults.csv*.
